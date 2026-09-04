package attendance_logs

import (
	"errors"
	"fmt"
	"time"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AttendanceLogService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *AttendanceLogService { return &AttendanceLogService{DB: db} }

func (s *AttendanceLogService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	params := make(map[string]interface{})
	for key, values := range ctx.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	params["al.deleted_at.isnull"] = ""
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		params["u.institution_id"] = institutionID
	}
	base := `select al.*, u.name as user_name, u.email as user_email, u.institution_id,
		lg.name as learning_group_name
		from attendance_logs al
		join users u on u.id = al.user_id
		left join learning_groups lg on lg.id = al.learning_group_id and lg.deleted_at is null`
	return helpers.BuildPaginatedQuery(ctx, s.DB, params, "attendance_logs", base, "", "", dto.SortBy)
}

func (s *AttendanceLogService) getByID(ctx *gin.Context, id string) (*models.AttendanceLog, error) {
	var data models.AttendanceLog
	query := s.DB.Joins("JOIN users ON users.id = attendance_logs.user_id").Preload("User").Preload("LearningGroup").
		Preload("AbsenceReason", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).Where("attendance_logs.id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("users.institution_id = ?", institutionID)
	}
	err := query.First(&data).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *AttendanceLogService) create(ctx *gin.Context, dto CreateDTO) (*models.AttendanceLog, error) {
	userID, learningGroupID, attendanceType, err := s.parseAndValidateContext(ctx, dto.UserID, dto.LearningGroupID, dto.Type)
	if err != nil {
		return nil, err
	}
	if dto.CheckOutAt != nil && dto.CheckInAt == nil {
		return nil, errors.New("check_in_at is required when check_out_at is provided")
	}
	if dto.CheckInAt != nil && dto.CheckOutAt != nil && dto.CheckOutAt.Before(*dto.CheckInAt) {
		return nil, errors.New("check_out_at cannot be earlier than check_in_at")
	}
	absenceReasonID, err := s.parseAbsenceReason(ctx, dto.AbsenceReasonID)
	if err != nil {
		return nil, err
	}
	data := models.AttendanceLog{
		UserID: userID, LearningGroupID: learningGroupID, Type: attendanceType,
		Status: models.AttendanceStatus(dto.Status), RequiresCheckOut: dto.RequiresCheckOut,
		AbsenceReasonID: absenceReasonID, AbsenceNote: dto.AbsenceNote,
		CheckInAt: dto.CheckInAt, CheckInLat: dto.CheckInLat, CheckInLong: dto.CheckInLong,
		CheckOutAt: dto.CheckOutAt, CheckOutLat: dto.CheckOutLat, CheckOutLong: dto.CheckOutLong,
	}
	if err := validateAttendanceState(data); err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if data.CheckInAt != nil {
			if err := lockAttendanceDay(tx, data.UserID, data.Type, data.LearningGroupID, *data.CheckInAt); err != nil {
				return err
			}
			if err := ensureNoDailyCheckIn(tx, data.UserID, data.Type, data.LearningGroupID, *data.CheckInAt, uuid.Nil); err != nil {
				return err
			}
		}
		return tx.Create(&data).Error
	}); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *AttendanceLogService) checkIn(ctx *gin.Context, dto CheckInDTO) (*models.AttendanceLog, error) {
	checkInAt := time.Now()
	if dto.CheckInAt != nil {
		checkInAt = *dto.CheckInAt
	}
	return s.create(ctx, CreateDTO{
		UserID: dto.UserID, LearningGroupID: dto.LearningGroupID, Type: dto.Type,
		Status: dto.Status, RequiresCheckOut: dto.RequiresCheckOut,
		CheckInAt: &checkInAt, CheckInLat: dto.LocationLat, CheckInLong: dto.LocationLong,
	})
}

func (s *AttendanceLogService) checkOut(ctx *gin.Context, id string, dto CheckOutDTO) (*models.AttendanceLog, error) {
	attendanceID, err := uuid.Parse(id)
	if err != nil {
		return nil, nil
	}
	var data models.AttendanceLog
	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Joins("JOIN users ON users.id = attendance_logs.user_id").Where("attendance_logs.id = ?", attendanceID)
		if institutionID := ctx.GetString("institution_id"); institutionID != "" {
			query = query.Where("users.institution_id = ?", institutionID)
		}
		if err := query.First(&data).Error; err != nil {
			return err
		}
		if data.CheckInAt == nil {
			return errors.New("attendance has not checked in")
		}
		if data.CheckOutAt != nil {
			return errors.New("attendance has already checked out")
		}
		checkOutAt := time.Now()
		if dto.CheckOutAt != nil {
			checkOutAt = *dto.CheckOutAt
		}
		if checkOutAt.Before(*data.CheckInAt) {
			return errors.New("check_out_at cannot be earlier than check_in_at")
		}
		data.CheckOutAt = &checkOutAt
		data.CheckOutLat = dto.LocationLat
		data.CheckOutLong = dto.LocationLong
		return tx.Save(&data).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *AttendanceLogService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.AttendanceLog, error) {
	data, err := s.getByID(ctx, id)
	if err != nil || data == nil {
		return data, err
	}
	userID, learningGroupID, attendanceType := data.UserID, data.LearningGroupID, data.Type
	userValue, groupValue, typeValue := userID.String(), nullableUUIDString(learningGroupID), string(attendanceType)
	if dto.UserID != nil {
		userValue = *dto.UserID
	}
	if dto.LearningGroupID != nil {
		groupValue = *dto.LearningGroupID
	}
	if dto.Type != nil {
		typeValue = *dto.Type
	}
	userID, learningGroupID, attendanceType, err = s.parseAndValidateContext(ctx, userValue, groupValue, typeValue)
	if err != nil {
		return nil, err
	}
	data.UserID, data.LearningGroupID, data.Type = userID, learningGroupID, attendanceType
	if dto.Status != nil {
		data.Status = models.AttendanceStatus(*dto.Status)
	}
	if dto.AbsenceReasonID != nil {
		absenceReasonID, err := s.parseAbsenceReason(ctx, *dto.AbsenceReasonID)
		if err != nil {
			return nil, err
		}
		data.AbsenceReasonID = absenceReasonID
	}
	if dto.AbsenceNote != nil {
		data.AbsenceNote = *dto.AbsenceNote
	}
	if dto.RequiresCheckOut != nil {
		data.RequiresCheckOut = *dto.RequiresCheckOut
	}
	if dto.CheckInAt != nil {
		data.CheckInAt = dto.CheckInAt
	}
	if dto.CheckInLat != nil {
		data.CheckInLat = dto.CheckInLat
	}
	if dto.CheckInLong != nil {
		data.CheckInLong = dto.CheckInLong
	}
	if dto.CheckOutAt != nil {
		data.CheckOutAt = dto.CheckOutAt
	}
	if dto.CheckOutLat != nil {
		data.CheckOutLat = dto.CheckOutLat
	}
	if dto.CheckOutLong != nil {
		data.CheckOutLong = dto.CheckOutLong
	}
	if data.CheckOutAt != nil && data.CheckInAt == nil {
		return nil, errors.New("check_in_at is required when check_out_at is provided")
	}
	if data.CheckInAt != nil && data.CheckOutAt != nil && data.CheckOutAt.Before(*data.CheckInAt) {
		return nil, errors.New("check_out_at cannot be earlier than check_in_at")
	}
	if err := validateAttendanceState(*data); err != nil {
		return nil, err
	}
	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if data.CheckInAt != nil {
			if err := lockAttendanceDay(tx, data.UserID, data.Type, data.LearningGroupID, *data.CheckInAt); err != nil {
				return err
			}
			if err := ensureNoDailyCheckIn(tx, data.UserID, data.Type, data.LearningGroupID, *data.CheckInAt, data.ID); err != nil {
				return err
			}
		}
		return tx.Save(data).Error
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *AttendanceLogService) parseAndValidateContext(ctx *gin.Context, userValue, groupValue, typeValue string) (uuid.UUID, *uuid.UUID, models.AttendanceType, error) {
	userID, err := uuid.Parse(userValue)
	if err != nil {
		return uuid.Nil, nil, "", errors.New("invalid user_id format")
	}
	if err := s.validateUser(ctx, userID); err != nil {
		return uuid.Nil, nil, "", err
	}
	var learningGroupID *uuid.UUID
	if groupValue != "" {
		parsed, err := uuid.Parse(groupValue)
		if err != nil {
			return uuid.Nil, nil, "", errors.New("invalid learning_group_id format")
		}
		learningGroupID = &parsed
	}
	attendanceType := models.AttendanceType(typeValue)
	if attendanceType == models.AttendanceTypeLearningGroup && learningGroupID == nil {
		return uuid.Nil, nil, "", errors.New("learning_group_id is required for learning_group attendance")
	}
	if learningGroupID != nil {
		if err := s.validateLearningGroupMember(ctx, *learningGroupID, userID); err != nil {
			return uuid.Nil, nil, "", err
		}
	}
	return userID, learningGroupID, attendanceType, nil
}

func (s *AttendanceLogService) validateLearningGroupMember(ctx *gin.Context, groupID, userID uuid.UUID) error {
	query := s.DB.Model(&models.LearningGroupMember{}).
		Joins("JOIN learning_groups ON learning_groups.id = learning_group_members.learning_group_id").
		Where("learning_group_members.learning_group_id = ? AND learning_group_members.user_id = ?", groupID, userID)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("learning_groups.institution_id = ?", institutionID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user is not a member of the learning group")
	}
	return nil
}

func (s *AttendanceLogService) parseAbsenceReason(ctx *gin.Context, value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, errors.New("invalid absence_reason_id format")
	}
	query := s.DB.Model(&models.AttendanceAbsenceReason{}).Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("attendance absence reason not found")
	}
	return &id, nil
}

func validateAttendanceState(data models.AttendanceLog) error {
	if data.Status == models.AttendanceStatusAbsent {
		if data.AbsenceReasonID == nil {
			return errors.New("absence_reason_id is required when status is absent")
		}
		if data.CheckInAt != nil || data.CheckOutAt != nil {
			return errors.New("absent attendance cannot have check-in or check-out")
		}
		if data.RequiresCheckOut {
			return errors.New("absent attendance cannot require check-out")
		}
		return nil
	}
	if data.AbsenceReasonID != nil || data.AbsenceNote != "" {
		return errors.New("absence reason is only allowed when status is absent")
	}
	return nil
}

func lockAttendanceDay(tx *gorm.DB, userID uuid.UUID, attendanceType models.AttendanceType, groupID *uuid.UUID, checkInAt time.Time) error {
	key := fmt.Sprintf("%s|%s|%s|%s", userID, attendanceType, nullableUUIDString(groupID), checkInAt.Format("2006-01-02"))
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func ensureNoDailyCheckIn(tx *gorm.DB, userID uuid.UUID, attendanceType models.AttendanceType, groupID *uuid.UUID, checkInAt time.Time, excludeID uuid.UUID) error {
	year, month, day := checkInAt.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, checkInAt.Location())
	query := tx.Model(&models.AttendanceLog{}).Where("user_id = ? AND type = ? AND check_in_at >= ? AND check_in_at < ?", userID, attendanceType, start, start.AddDate(0, 0, 1))
	if groupID == nil {
		query = query.Where("learning_group_id IS NULL")
	} else {
		query = query.Where("learning_group_id = ?", *groupID)
	}
	if excludeID != uuid.Nil {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("user has already checked in for this attendance context and date")
	}
	return nil
}

func nullableUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func (s *AttendanceLogService) archive(ctx *gin.Context, id string) (bool, error) {
	return s.remove(ctx, id, false)
}

func (s *AttendanceLogService) delete(ctx *gin.Context, id string) (bool, error) {
	return s.remove(ctx, id, true)
}

func (s *AttendanceLogService) remove(ctx *gin.Context, id string, permanent bool) (bool, error) {
	var data models.AttendanceLog
	lookup := s.DB.Joins("JOIN users ON users.id = attendance_logs.user_id").Where("attendance_logs.id = ?", id)
	if permanent {
		lookup = lookup.Unscoped()
	}
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		lookup = lookup.Where("users.institution_id = ?", institutionID)
	}
	if err := lookup.First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	remove := s.DB.WithContext(ctx.Request.Context())
	if permanent {
		remove = remove.Unscoped()
	}
	result := remove.Delete(&data)
	return result.RowsAffected > 0, result.Error
}

func (s *AttendanceLogService) validateUser(ctx *gin.Context, userID uuid.UUID) error {
	query := s.DB.Model(&models.User{}).Where("id = ?", userID)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user not found")
	}
	return nil
}
