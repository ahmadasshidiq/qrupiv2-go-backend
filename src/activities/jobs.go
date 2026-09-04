package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kafkalib "clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/libs/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ActivityBulkRequestedEvent = "activity.bulk.requested"

var activityOutboxWake = make(chan struct{}, 1)

type activityBulkEventData struct {
	JobID string `json:"job_id"`
}

func (s *ActivityService) enqueueBulk(ctx context.Context, dto BulkCreateDTO, institutionValue string) (*BulkJobResult, error) {
	payload, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("encode bulk activity payload: %w", err)
	}
	recordedUserID, err := uuid.Parse(dto.RecordedUserID)
	if err != nil {
		return nil, err
	}
	institutionID, err := parseOptionalUUID(institutionValue)
	if err != nil {
		return nil, fmt.Errorf("invalid institution context: %w", err)
	}
	job := models.ActivityBulkJob{
		InstitutionID: institutionID, RecordedUserID: recordedUserID,
		Status: models.ActivityBulkJobPending, TotalData: len(dto.Activities), Payload: datatypes.JSON(payload),
	}

	institutionKey := "system"
	if institutionID != nil {
		institutionKey = institutionID.String()
	}
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		event, err := kafkalib.NewEnvelope(ActivityBulkRequestedEvent, "activity-service", institutionKey, job.ID.String(), activityBulkEventData{JobID: job.ID.String()})
		if err != nil {
			return err
		}
		eventPayload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		return tx.Create(&models.ActivityOutboxEvent{
			AggregateID: job.ID, EventType: ActivityBulkRequestedEvent, Payload: datatypes.JSON(eventPayload),
		}).Error
	}); err != nil {
		return nil, err
	}
	wakeActivityOutbox()
	return &BulkJobResult{
		JobID: job.ID.String(), Status: job.Status, TotalData: job.TotalData,
		Message:                    "Data Anda sedang diproses, mohon menunggu maksimal 5 menit",
		EstimatedCompletionMinutes: 5, RetryAfterSeconds: 3,
	}, nil
}

func (s *ActivityService) getBulkJob(id, institutionValue string) (*models.ActivityBulkJob, error) {
	jobID, err := uuid.Parse(id)
	if err != nil {
		return nil, nil
	}
	query := s.DB.Where("id = ?", jobID)
	if institutionValue != "" {
		institutionID, err := uuid.Parse(institutionValue)
		if err != nil {
			return nil, err
		}
		query = query.Where("institution_id = ?", institutionID)
	}
	var job models.ActivityBulkJob
	if err := query.First(&job).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &job, nil
}

func StartActivityOutboxDispatcher(ctx context.Context, db *gorm.DB, producer kafkalib.Producer, topic string, logger *slog.Logger) {
	// Check once on startup to recover events committed before a previous process stopped.
	pending := true
	retryDelay := time.Second
	for {
		if pending {
			err := dispatchAllActivityOutbox(ctx, db, producer, topic)
			if err == nil {
				pending = false
				retryDelay = time.Second
			} else if ctx.Err() == nil {
				logger.ErrorContext(ctx, "activity outbox dispatch failed", "error", err, "retry_after", retryDelay)
				timer := time.NewTimer(retryDelay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-activityOutboxWake:
					timer.Stop()
				case <-timer.C:
				}
				if retryDelay < 30*time.Second {
					retryDelay *= 2
				}
				continue
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-activityOutboxWake:
			pending = true
		}
	}
}

func wakeActivityOutbox() {
	select {
	case activityOutboxWake <- struct{}{}:
	default:
	}
}

func dispatchAllActivityOutbox(ctx context.Context, db *gorm.DB, producer kafkalib.Producer, topic string) error {
	for {
		processed, err := dispatchActivityOutbox(ctx, db, producer, topic)
		if err != nil {
			return err
		}
		if processed == 0 {
			return nil
		}
	}
}

func dispatchActivityOutbox(ctx context.Context, db *gorm.DB, producer kafkalib.Producer, topic string) (int, error) {
	var events []models.ActivityOutboxEvent
	if err := db.WithContext(ctx).Where("published_at IS NULL").Order("created_at").Limit(100).Find(&events).Error; err != nil {
		return 0, err
	}
	for _, outboxEvent := range events {
		var event kafkalib.Envelope
		if err := json.Unmarshal(outboxEvent.Payload, &event); err != nil {
			if markErr := markOutboxFailure(ctx, db, outboxEvent.ID, err); markErr != nil {
				return 0, markErr
			}
			return 0, err
		}
		if err := producer.Publish(ctx, topic, event); err != nil {
			if markErr := markOutboxFailure(ctx, db, outboxEvent.ID, err); markErr != nil {
				return 0, markErr
			}
			return 0, err
		}
		now := time.Now().UTC()
		if err := db.WithContext(ctx).Model(&models.ActivityOutboxEvent{}).Where("id = ? AND published_at IS NULL", outboxEvent.ID).Updates(map[string]interface{}{
			"published_at": now, "last_error": "",
		}).Error; err != nil {
			return 0, err
		}
	}
	return len(events), nil
}

func markOutboxFailure(ctx context.Context, db *gorm.DB, id uuid.UUID, cause error) error {
	return db.WithContext(ctx).Model(&models.ActivityOutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{
		"attempts": gorm.Expr("attempts + 1"), "last_error": cause.Error(),
	}).Error
}

func HandleActivityBulkEvent(db *gorm.DB, maxAttempts int) kafkalib.Handler {
	if maxAttempts < 1 {
		maxAttempts = 3
	}
	return func(ctx context.Context, event kafkalib.Envelope) error {
		if event.EventType != ActivityBulkRequestedEvent {
			return nil
		}
		var data activityBulkEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return err
		}
		jobID, err := uuid.Parse(data.JobID)
		if err != nil {
			return err
		}
		return processActivityBulkJob(ctx, db, jobID, maxAttempts)
	}
}

func processActivityBulkJob(ctx context.Context, db *gorm.DB, jobID uuid.UUID, maxAttempts int) error {
	result := db.WithContext(ctx).Model(&models.ActivityBulkJob{}).Where("id = ? AND status IN ?", jobID, []models.ActivityBulkJobStatus{
		models.ActivityBulkJobPending, models.ActivityBulkJobProcessing,
	}).UpdateColumn("attempts", gorm.Expr("attempts + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	now := time.Now().UTC()
	statusResult := db.WithContext(ctx).Model(&models.ActivityBulkJob{}).Where("id = ? AND status IN ?", jobID, []models.ActivityBulkJobStatus{
		models.ActivityBulkJobPending, models.ActivityBulkJobProcessing,
	}).Updates(map[string]interface{}{
		"status":        models.ActivityBulkJobProcessing,
		"started_at":    gorm.Expr("COALESCE(started_at, ?)", now),
		"error_message": "",
	})
	if statusResult.Error != nil {
		return statusResult.Error
	}
	if statusResult.RowsAffected == 0 {
		return nil
	}

	processingError := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job models.ActivityBulkJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, "id = ?", jobID).Error; err != nil {
			return err
		}
		if job.Status == models.ActivityBulkJobCompleted || job.Status == models.ActivityBulkJobFailed {
			return nil
		}
		var dto BulkCreateDTO
		if err := json.Unmarshal(job.Payload, &dto); err != nil {
			return &BulkValidationError{Index: -1, Field: "payload", Message: err.Error()}
		}
		result, err := (&ActivityService{DB: tx}).createBulk(dto)
		if err != nil {
			return err
		}
		completedAt := time.Now().UTC()
		return tx.Model(&job).Updates(map[string]interface{}{
			"status": models.ActivityBulkJobCompleted, "processed_data": result.Count,
			"failed_data": 0, "completed_at": completedAt, "error_message": "",
		}).Error
	})
	if processingError == nil {
		return nil
	}

	var job models.ActivityBulkJob
	if err := db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return err
	}
	var validationError *BulkValidationError
	isValidationError := errors.As(processingError, &validationError)
	terminal := isValidationError || job.Attempts >= maxAttempts
	if !terminal {
		return processingError
	}
	completedAt := time.Now().UTC()
	if err := db.WithContext(ctx).Model(&job).Updates(map[string]interface{}{
		"status": models.ActivityBulkJobFailed, "failed_data": job.TotalData,
		"completed_at": completedAt, "error_message": processingError.Error(),
	}).Error; err != nil {
		return err
	}
	if !isValidationError {
		return processingError
	}
	return nil
}
