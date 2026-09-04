package activitycategories

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityCategoryService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *ActivityCategoryService { return &ActivityCategoryService{DB: db} }
func (s *ActivityCategoryService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	p := map[string]interface{}{}
	for k, v := range ctx.Request.URL.Query() {
		if len(v) > 0 {
			p[k] = v[0]
		}
	}
	p["activity_categories.deleted_at.isnull"] = ""
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		p["activity_categories.institution_id"] = institutionID
	}
	return helpers.BuildPaginatedQuery(ctx, s.DB, p, "activity_categories", "", "", "", dto.SortBy)
}
func (s *ActivityCategoryService) getByID(ctx *gin.Context, id string) (*models.ActivityCategory, error) {
	var d models.ActivityCategory
	query := s.DB.Preload("Items")
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	e := query.First(&d, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &d, e
}
func (s *ActivityCategoryService) create(ctx *gin.Context, dto CreateDTO) (*models.ActivityCategory, error) {
	d := models.ActivityCategory{InstitutionID: institutionID(ctx), Name: dto.Name, Description: dto.Description, Color: dto.Color}
	e := s.DB.Create(&d).Error
	return &d, e
}
func (s *ActivityCategoryService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.ActivityCategory, error) {
	var d models.ActivityCategory
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
	if dto.Name != nil {
		d.Name = *dto.Name
	}
	if dto.Description != nil {
		d.Description = *dto.Description
	}
	if dto.Color != nil {
		d.Color = *dto.Color
	}
	e = s.DB.Save(&d).Error
	return &d, e
}
func (s *ActivityCategoryService) archive(ctx *gin.Context, id string) (bool, error) {
	query := s.DB.Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	r := query.Delete(&models.ActivityCategory{})
	return r.RowsAffected > 0, r.Error
}
func (s *ActivityCategoryService) delete(ctx *gin.Context, id string) (bool, error) {
	query := s.DB.Unscoped().Where("id = ?", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	r := query.Delete(&models.ActivityCategory{})
	return r.RowsAffected > 0, r.Error
}

func institutionID(ctx *gin.Context) *uuid.UUID {
	id, err := uuid.Parse(ctx.GetString("institution_id"))
	if err != nil {
		return nil
	}
	return &id
}
