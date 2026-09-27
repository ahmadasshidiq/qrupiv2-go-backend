package legal_documents

import (
	"errors"
	"fmt"
	"time"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{DB: db} }

func (s *Service) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	return helpers.BuildPaginatedQuery(
		ctx,
		s.DB,
		params,
		"legal_documents",
		"select ld.* from legal_documents ld",
		"",
		"",
		dto.SortBy,
	)
}
func (s *Service) getByID(id string) (*models.LegalDocument, error) {
	var d models.LegalDocument
	e := s.DB.First(&d, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &d, e
}
func parseDate(v string) (time.Time, error) {
	t, e := time.Parse(time.RFC3339, v)
	if e != nil {
		return time.Time{}, fmt.Errorf("effective_date must be RFC3339")
	}
	return t, e
}

func (s *Service) create(dto CreateDTO) (*models.LegalDocument, error) {
	t, e := parseDate(dto.EffectiveDate)
	if e != nil {
		return nil, e
	}
	d := &models.LegalDocument{ID: uuid.New(), Slug: dto.Slug, Title: dto.Title, Content: dto.Content, Version: dto.Version, Status: dto.Status, EffectiveDate: t}
	return d, s.DB.Create(d).Error
}
func (s *Service) update(id string, dto UpdateDTO) (*models.LegalDocument, error) {
	d, e := s.getByID(id)
	if e != nil || d == nil {
		return d, e
	}
	updates := map[string]interface{}{}
	if dto.Slug != nil {
		updates["slug"] = *dto.Slug
	}
	if dto.Title != nil {
		updates["title"] = *dto.Title
	}
	if dto.Content != nil {
		updates["content"] = *dto.Content
	}
	if dto.Version != nil {
		updates["version"] = *dto.Version
	}
	if dto.Status != nil {
		updates["status"] = *dto.Status
	}
	if dto.EffectiveDate != nil {
		t, x := parseDate(*dto.EffectiveDate)
		if x != nil {
			return nil, x
		}
		updates["effective_date"] = t
	}
	e = s.DB.Model(d).Updates(updates).Error
	return d, e
}
func (s *Service) delete(id string) (bool, error) {
	r := s.DB.Delete(&models.LegalDocument{}, "id = ?", id)
	return r.RowsAffected > 0, r.Error
}
