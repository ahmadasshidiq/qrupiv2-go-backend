package notifications

import (
	"clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"log/slog"
	"time"
)

type Service struct {
	DB     *gorm.DB
	Sender *notif.FCMSender
	Logger *slog.Logger
}

func (s *Service) StartScheduler(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.runScheduled(ctx, now)
		}
	}
}

func (s *Service) runScheduled(ctx context.Context, now time.Time) {
	var rows []models.Announcement
	s.DB.WithContext(ctx).Where("status IN ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND deleted_at IS NULL", []string{"scheduled", "published"}, now).Limit(100).Find(&rows)
	for _, a := range rows {
		var ids []uuid.UUID
		switch a.Audience {
		case models.AnnouncementAudienceUser:
			if a.TargetID != nil {
				ids = []uuid.UUID{*a.TargetID}
			}
		case models.AnnouncementAudienceLearningGroup:
			if a.TargetID != nil {
				s.DB.Table("learning_group_members").Where("learning_group_id=? AND role_in_group='student' AND deleted_at IS NULL", *a.TargetID).Pluck("user_id", &ids)
			}
		case models.AnnouncementAudienceInstitution:
			s.DB.Model(&models.User{}).Where("institution_id=? AND deleted_at IS NULL", a.InstitutionID).Pluck("id", &ids)
		default:
			s.DB.Model(&models.User{}).Where("deleted_at IS NULL").Pluck("id", &ids)
		}
		recipients := make([]string, 0, len(ids))
		for _, id := range ids {
			recipients = append(recipients, id.String())
		}
		_ = s.Persist(ctx, "announcement.created", derefUUID(a.InstitutionID), recipients, a.Title, a.Message, a.ID.String(), "", "", "", nil)
		next, ok := nextOccurrence(a.NextRunAt, a.RepeatType, a.RepeatUntil)
		if ok {
			s.DB.Model(&a).Updates(map[string]any{"next_run_at": next, "status": "scheduled"})
		} else {
			s.DB.Model(&a).Updates(map[string]any{"next_run_at": nil, "status": "sent", "published_at": now})
		}
	}
}

func derefUUID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
func nextOccurrence(at *time.Time, repeat string, until *time.Time) (*time.Time, bool) {
	if at == nil || repeat == "none" {
		return nil, false
	}
	n := *at
	switch repeat {
	case "daily":
		n = n.AddDate(0, 0, 1)
	case "weekly":
		n = n.AddDate(0, 0, 7)
	case "monthly":
		n = n.AddDate(0, 1, 0)
	default:
		return nil, false
	}
	if until != nil && n.After(*until) {
		return nil, false
	}
	return &n, true
}

func NewService(db *gorm.DB, sender *notif.FCMSender, logger *slog.Logger) *Service {
	return &Service{db, sender, logger}
}

func (s *Service) Persist(ctx context.Context, eventType, institution string, recipients []string, title, message, entityID, deeplink, webURL, mobileRoute string, data map[string]any) error {
	institutionID, _ := uuid.Parse(institution)
	eid, _ := uuid.Parse(entityID)
	raw, _ := json.Marshal(data)
	for _, recipient := range recipients {
		userID, err := uuid.Parse(recipient)
		if err != nil {
			continue
		}
		var existing int64
		s.DB.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ? AND event_type = ? AND entity_id = ? AND deleted_at IS NULL", userID, eventType, eid).Count(&existing)
		if existing > 0 {
			continue
		}
		n := models.Notification{UserID: userID, EventType: eventType, Title: title, Message: message, Deeplink: deeplink, WebURL: webURL, MobileRoute: mobileRoute, Data: datatypes.JSON(raw)}
		if institutionID != uuid.Nil {
			n.InstitutionID = &institutionID
		}
		if eid != uuid.Nil {
			n.EntityID = &eid
		}
		if err := s.DB.WithContext(ctx).Create(&n).Error; err != nil {
			return err
		}
		if s.Sender != nil {
			var devices []models.NotificationDevice
			s.DB.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Find(&devices)
			for _, d := range devices {
				_, _ = s.Sender.Send(ctx, d.Token, title, message, deeplink, webURL, mobileRoute, entityID)
			}
		}
	}
	return nil
}

func (s *Service) HandleEvent(ctx context.Context, event kafka.Envelope) error {
	var payload EventDataDTO
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}
	if err := s.Persist(ctx, event.EventType, event.InstitutionID, payload.RecipientIDs, payload.Title, payload.Message, payload.EntityID, payload.Deeplink, payload.WebURL, payload.MobileRoute, payload.Data); err != nil {
		return err
	}
	if s.Logger != nil {
		s.Logger.InfoContext(ctx, "notification event processed", "event_type", event.EventType, "event_id", event.EventID, "recipients", len(payload.RecipientIDs))
	}
	return nil
}
