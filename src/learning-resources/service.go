package learning_resources

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/stores"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LearningResourceService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *LearningResourceService { return &LearningResourceService{DB: db} }

func (s *LearningResourceService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	params := make(map[string]interface{})
	for key, values := range ctx.Request.URL.Query() {
		if len(values) > 0 {
			if key == "learning_group_id" {
				params["lrg.learning_group_id"] = values[0]
				continue
			}
			params[key] = values[0]
		}
	}
	params["lr.deleted_at.isnull"] = ""
	baseQuery := `select lr.*, u.name as uploaded_user_name, i.name as institution_name,
		COALESCE(array_agg(lg.id) FILTER (WHERE lg.id IS NOT NULL), '{}') as learning_group_ids,
		COALESCE(string_agg(lg.name, ', ' ORDER BY lg.name) FILTER (WHERE lg.id IS NOT NULL), '') as learning_group_names
		from learning_resources lr
		join users u on u.id = lr.uploaded_user_id
		left join institutions i on i.id = u.institution_id
		left join learning_resource_groups lrg on lrg.learning_resource_id = lr.id
		left join learning_groups lg on lg.id = lrg.learning_group_id and lg.deleted_at is null`
	return helpers.BuildPaginatedQuery(ctx, s.DB, params, "learning_resources", baseQuery, "group by lr.id, u.name, i.name", "", dto.SortBy)
}

func (s *LearningResourceService) getByID(id string) (*models.LearningResource, error) {
	var data models.LearningResource
	err := s.DB.Preload("LearningGroups").Preload("UploadedUser").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	data.LearningGroupIDs = learningGroupIDs(data.LearningGroups)
	return &data, err
}

func (s *LearningResourceService) create(ctx *gin.Context, dto CreateDTO) (*models.LearningResource, error) {
	groups, err := s.resolveLearningGroups(dto.LearningGroupIDs)
	if err != nil {
		return nil, err
	}
	uploadedUserID, institutionID, err := s.resolveUploader(dto.UploadedUserID)
	if err != nil {
		return nil, err
	}
	data := models.LearningResource{
		UploadedUserID: uploadedUserID, Title: dto.Title, Description: dto.Description,
		Type: models.LearningResourceType(dto.Type),
	}
	if dto.Type != "file" {
		data.FileURL = dto.FileURL
	}

	file, header, fileErr := ctx.Request.FormFile("file")
	if fileErr == nil {
		defer file.Close()
		publicURL, err := stores.UploadToMinio(file, os.Getenv("MINIO_PRODUCT_BUCKET"), institutionID, header.Filename, header.Header.Get("Content-Type"), header.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to upload learning resource: %w", err)
		}
		data.FileURL = publicURL
	} else if fileErr != http.ErrMissingFile {
		return nil, fileErr
	}

	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&data).Error; err != nil {
			return err
		}
		return createResourceGroupLinks(tx, data.ID, groups)
	})
	if err != nil {
		if data.FileURL != "" && fileErr == nil {
			stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), data.FileURL)
		}
		return nil, err
	}
	data.LearningGroups = groups
	data.LearningGroupIDs = learningGroupIDs(groups)
	return &data, nil
}

func (s *LearningResourceService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.LearningResource, error) {
	var data models.LearningResource
	if err := s.DB.First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	oldFileURL := data.FileURL
	var groups []models.LearningGroup
	if dto.LearningGroupIDs != nil {
		var err error
		groups, err = s.resolveLearningGroups(dto.LearningGroupIDs)
		if err != nil {
			return nil, err
		}
	}
	if dto.UploadedUserID != nil {
		uploadedUserID, _, err := s.resolveUploader(*dto.UploadedUserID)
		if err != nil {
			return nil, err
		}
		data.UploadedUserID = uploadedUserID
	}
	_, institutionID, err := s.resolveUploader(data.UploadedUserID.String())
	if err != nil {
		return nil, err
	}
	if dto.Title != nil {
		data.Title = *dto.Title
	}
	if dto.Description != nil {
		data.Description = *dto.Description
	}
	if dto.Type != nil {
		data.Type = models.LearningResourceType(*dto.Type)
	}
	if dto.FileURL != nil {
		data.FileURL = *dto.FileURL
	}

	file, header, fileErr := ctx.Request.FormFile("file")
	var uploadedURL string
	if fileErr == nil {
		defer file.Close()
		uploadedURL, err = stores.UploadToMinio(file, os.Getenv("MINIO_PRODUCT_BUCKET"), institutionID, header.Filename, header.Header.Get("Content-Type"), header.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to upload learning resource: %w", err)
		}
		data.FileURL = uploadedURL
	} else if fileErr != http.ErrMissingFile {
		return nil, fileErr
	}

	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&data).Error; err != nil {
			return err
		}
		if dto.LearningGroupIDs == nil {
			return nil
		}
		if err := tx.Where("learning_resource_id = ?", data.ID).Delete(&models.LearningResourceGroup{}).Error; err != nil {
			return err
		}
		return createResourceGroupLinks(tx, data.ID, groups)
	})
	if err != nil {
		if uploadedURL != "" {
			stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), uploadedURL)
		}
		return nil, err
	}
	if uploadedURL != "" && oldFileURL != "" && oldFileURL != uploadedURL {
		stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), oldFileURL)
	}
	if dto.LearningGroupIDs != nil {
		data.LearningGroups = groups
	} else {
		_ = s.DB.Model(&data).Association("LearningGroups").Find(&data.LearningGroups)
	}
	data.LearningGroupIDs = learningGroupIDs(data.LearningGroups)
	return &data, nil
}

func learningGroupIDs(groups []models.LearningGroup) []uuid.UUID {
	ids := make([]uuid.UUID, len(groups))
	for index := range groups {
		ids[index] = groups[index].ID
	}
	return ids
}

func (s *LearningResourceService) resolveLearningGroups(values []string) ([]models.LearningGroup, error) {
	if len(values) == 0 {
		return nil, errors.New("learning_group_ids must contain at least one learning group")
	}
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		for _, candidate := range strings.Split(value, ",") {
			id, err := uuid.Parse(strings.TrimSpace(candidate))
			if err != nil {
				return nil, errors.New("invalid learning_group_ids format")
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	var groups []models.LearningGroup
	if err := s.DB.Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) != len(ids) {
		return nil, errors.New("one or more learning groups not found")
	}
	return groups, nil
}

func (s *LearningResourceService) resolveUploader(value string) (uuid.UUID, string, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, "", errors.New("invalid uploaded_user_id format")
	}
	var user models.User
	if err := s.DB.Select("id", "institution_id").First(&user, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, "", errors.New("uploaded user not found")
	} else if err != nil {
		return uuid.Nil, "", err
	}
	institutionID := "system"
	if user.InstitutionID != nil {
		institutionID = user.InstitutionID.String()
	}
	return id, institutionID, nil
}

func createResourceGroupLinks(tx *gorm.DB, resourceID uuid.UUID, groups []models.LearningGroup) error {
	links := make([]models.LearningResourceGroup, len(groups))
	for index, group := range groups {
		links[index] = models.LearningResourceGroup{LearningResourceID: resourceID, LearningGroupID: group.ID}
	}
	return tx.Create(&links).Error
}

func (s *LearningResourceService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.LearningResource{})
	return result.RowsAffected > 0, result.Error
}

func (s *LearningResourceService) delete(id string) (bool, error) {
	var data models.LearningResource
	if err := s.DB.Unscoped().First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("learning_resource_id = ?", data.ID).Delete(&models.LearningResourceGroup{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&data).Error
	}); err != nil {
		return false, err
	}
	if data.FileURL != "" {
		stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), data.FileURL)
	}
	return true, nil
}
