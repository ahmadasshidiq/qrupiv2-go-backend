package activityitems

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityItemService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *ActivityItemService { return &ActivityItemService{DB: db} }
func (s *ActivityItemService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	p := map[string]interface{}{}
	for k, v := range ctx.Request.URL.Query() {
		if len(v) > 0 {
			p[k] = v[0]
		}
	}
	p["ai.deleted_at.isnull"] = ""
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		p["ai.institution_id"] = institutionID
	}
	base := `select ai.*, ac.name as category_name from activity_items ai
		left join activity_categories ac on ac.id = ai.category_id and ac.deleted_at is null`
	return helpers.BuildPaginatedQuery(ctx, s.DB, p, "activity_items", base, "", "", dto.SortBy)
}
func (s *ActivityItemService) getByID(ctx *gin.Context, id string) (*models.ActivityItem, error) {
	var d models.ActivityItem
	query := s.DB.Preload("Category")
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	e := query.First(&d, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &d, e
}
func (s *ActivityItemService) create(ctx *gin.Context, dto CreateDTO) (*models.ActivityItem, error) {
	categoryID, e := s.parseCategoryID(ctx, dto.CategoryID)
	if e != nil {
		return nil, e
	}
	periodType := models.ActivityLimitType(dto.PeriodType)
	if periodType == "" {
		periodType = models.ActivityLimitNone
	}
	if err := validatePeriodLimit(dto.PeriodLimit, periodType); err != nil {
		return nil, err
	}
	d := models.ActivityItem{InstitutionID: institutionID(ctx), CategoryID: categoryID, Name: dto.Name, Description: dto.Description, Type: models.ActivityItemType(dto.Type), PointValue: dto.PointValue, DailyLimit: dto.DailyLimit, PeriodLimit: dto.PeriodLimit, PeriodType: periodType, IsSendNotif: dto.IsSendNotif, Color: dto.Color}
	e = s.DB.Create(&d).Error
	return &d, e
}
func (s *ActivityItemService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.ActivityItem, error) {
	var d models.ActivityItem
	query := s.DB
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	e := query.First(&d, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if dto.CategoryID != nil {
		categoryID, err := s.parseCategoryID(ctx, *dto.CategoryID)
		if err != nil {
			return nil, err
		}
		d.CategoryID = categoryID
	}
	if dto.Name != nil {
		d.Name = *dto.Name
	}
	if dto.Description != nil {
		d.Description = *dto.Description
	}
	if dto.Type != nil {
		d.Type = models.ActivityItemType(*dto.Type)
	}
	if dto.PointValue != nil {
		d.PointValue = *dto.PointValue
	}
	if dto.DailyLimit != nil {
		d.DailyLimit = *dto.DailyLimit
	}
	if dto.PeriodLimit != nil {
		d.PeriodLimit = *dto.PeriodLimit
	}
	if dto.PeriodType != nil {
		d.PeriodType = models.ActivityLimitType(*dto.PeriodType)
	}
	if err := validatePeriodLimit(d.PeriodLimit, d.PeriodType); err != nil {
		return nil, err
	}
	if dto.IsSendNotif != nil {
		d.IsSendNotif = *dto.IsSendNotif
	}
	if dto.Color != nil {
		d.Color = *dto.Color
	}
	e = s.DB.Save(&d).Error
	return &d, e
}

func validatePeriodLimit(limit int, periodType models.ActivityLimitType) error {
	if limit > 0 && periodType == models.ActivityLimitNone {
		return errors.New("period_type is required when period_limit is greater than 0")
	}
	if limit == 0 && periodType != models.ActivityLimitNone {
		return fmt.Errorf("period_limit must be greater than 0 when period_type is %s", periodType)
	}
	return nil
}

func (s *ActivityItemService) parseCategoryID(ctx *gin.Context, value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, errors.New("invalid category_id format")
	}
	if err := s.ensureCategory(ctx, id); err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *ActivityItemService) ensureCategory(ctx *gin.Context, id uuid.UUID) error {
	var count int64
	query := s.DB.Model(&models.ActivityCategory{}).Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("activity category not found")
	}
	return nil
}
func (s *ActivityItemService) archive(ctx *gin.Context, id string) (bool, error) {
	query := s.DB.Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	r := query.Delete(&models.ActivityItem{})
	return r.RowsAffected > 0, r.Error
}
func (s *ActivityItemService) delete(ctx *gin.Context, id string) (bool, error) {
	query := s.DB.Unscoped().Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	r := query.Delete(&models.ActivityItem{})
	return r.RowsAffected > 0, r.Error
}

func institutionID(ctx *gin.Context) *uuid.UUID {
	id, err := uuid.Parse(ctx.GetString("institution_id"))
	if err != nil {
		return nil
	}
	return &id
}
