package attendance_absence_reasons

import (
	"errors"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AttendanceAbsenceReasonService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *AttendanceAbsenceReasonService {
	return &AttendanceAbsenceReasonService{DB: db}
}

func (s *AttendanceAbsenceReasonService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	params := make(map[string]interface{})
	for key, values := range ctx.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	params["aar.deleted_at.isnull"] = ""
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		params["aar.institution_id"] = institutionID
	}
	base := `select aar.* from attendance_absence_reasons aar`
	return helpers.BuildPaginatedQuery(ctx, s.DB, params, "attendance_absence_reasons", base, "", "", dto.SortBy)
}

func (s *AttendanceAbsenceReasonService) getByID(ctx *gin.Context, id string) (*models.AttendanceAbsenceReason, error) {
	query := s.DB.Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	var data models.AttendanceAbsenceReason
	if err := query.First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *AttendanceAbsenceReasonService) create(ctx *gin.Context, dto CreateDTO) (*models.AttendanceAbsenceReason, error) {
	institutionID, err := institutionIDForCreate(ctx, dto.InstitutionID)
	if err != nil {
		return nil, err
	}
	data := models.AttendanceAbsenceReason{InstitutionID: institutionID, Name: dto.Name, Description: dto.Description}
	if err := s.DB.WithContext(ctx.Request.Context()).Create(&data).Error; err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *AttendanceAbsenceReasonService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.AttendanceAbsenceReason, error) {
	data, err := s.getByID(ctx, id)
	if err != nil || data == nil {
		return data, err
	}
	if dto.Name != nil {
		data.Name = *dto.Name
	}
	if dto.Description != nil {
		data.Description = *dto.Description
	}
	if err := s.DB.WithContext(ctx.Request.Context()).Save(data).Error; err != nil {
		return nil, err
	}
	return data, nil
}

func (s *AttendanceAbsenceReasonService) archive(ctx *gin.Context, id string) (bool, error) {
	data, err := s.getByID(ctx, id)
	if err != nil || data == nil {
		return false, err
	}
	result := s.DB.WithContext(ctx.Request.Context()).Delete(data)
	return result.RowsAffected > 0, result.Error
}

func (s *AttendanceAbsenceReasonService) delete(ctx *gin.Context, id string) (bool, error) {
	reasonID, err := uuid.Parse(id)
	if err != nil {
		return false, nil
	}
	query := s.DB.Unscoped().Where("id = ?", reasonID)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	var data models.AttendanceAbsenceReason
	if err := query.First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	var references int64
	if err := s.DB.Model(&models.AttendanceLog{}).Where("absence_reason_id = ?", reasonID).Count(&references).Error; err != nil {
		return false, err
	}
	if references > 0 {
		return false, errors.New("attendance absence reason is already used and cannot be permanently deleted")
	}
	result := s.DB.WithContext(ctx.Request.Context()).Unscoped().Delete(&data)
	return result.RowsAffected > 0, result.Error
}

func institutionIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
	value := ctx.GetString("institution_id")
	if value == "" {
		return uuid.Nil, errors.New("institution_id is required")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, errors.New("invalid institution_id context")
	}
	return id, nil
}

func institutionIDForCreate(ctx *gin.Context, requested string) (uuid.UUID, error) {
	if value := ctx.GetString("institution_id"); value != "" {
		return uuid.Parse(value)
	}
	if requested == "" {
		return uuid.Nil, errors.New("institution_id is required")
	}
	id, err := uuid.Parse(requested)
	if err != nil {
		return uuid.Nil, errors.New("invalid institution_id format")
	}
	return id, nil
}
