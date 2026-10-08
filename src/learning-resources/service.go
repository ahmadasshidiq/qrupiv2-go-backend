package learning_resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"
	"clasenna-go-backend/libs/stores"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxLearningResourceFileSize int64 = 8 * 1024 * 1024

type LearningResourceService struct {
	DB       *gorm.DB
	Notifier notif.Publisher
}

func NewService(db *gorm.DB, publishers ...notif.Publisher) *LearningResourceService {
	s := &LearningResourceService{DB: db}
	if len(publishers) > 0 {
		s.Notifier = publishers[0]
	}
	return s
}

func uploadLearningResourceFiles(ctx *gin.Context, institutionID string) ([]string, error) {
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			return nil, http.ErrMissingFile
		}
		return nil, err
	}
	fileHeaders := ctx.Request.MultipartForm.File["file"]

	urls := make([]string, 0, len(fileHeaders))
	for _, header := range fileHeaders {
		if header.Size > maxLearningResourceFileSize {
			deleteLearningResourceFiles(urls)
			return nil, fmt.Errorf("file %q exceeds maximum size of 8 MB", header.Filename)
		}
		file, err := header.Open()
		if err != nil {
			deleteLearningResourceFiles(urls)
			return nil, err
		}
		publicURL, err := stores.UploadToMinio(file, os.Getenv("MINIO_PRODUCT_BUCKET"), institutionID, header.Filename, header.Header.Get("Content-Type"), header.Size)
		file.Close()
		if err != nil {
			deleteLearningResourceFiles(urls)
			return nil, fmt.Errorf("failed to upload learning resource: %w", err)
		}
		urls = append(urls, publicURL)
	}
	return urls, nil
}

func deleteLearningResourceFiles(urls []string) {
	publicPrefix := strings.TrimSuffix(stores.MinioPublicURL, "/") + "/" + os.Getenv("MINIO_PRODUCT_BUCKET") + "/"
	for _, url := range urls {
		if url != "" && strings.HasPrefix(url, publicPrefix) {
			stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), url)
		}
	}
}

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
	var role string
	if err := s.DB.Table("roles").Select("lower(replace(name, '-', '_'))").Where("id = ?", ctx.GetString("role_id")).Scan(&role).Error; err != nil {
		return nil, err
	}
	// The authenticated role alone is not sufficient to identify a student
	// viewer. Institution accounts can have an institution scope and may use a
	// role configuration that resolves to the student role, but they must still
	// be able to see all resources in that institution.
	var userType string
	if err := s.DB.Table("users").Select("type").Where("id = ?", ctx.GetString("user_id")).Scan(&userType).Error; err != nil {
		return nil, err
	}
	isStudentViewer := helpers.IsRole(role, "student") && strings.EqualFold(userType, "student")
	userID := ctx.GetString("user_id")
	if isStudentViewer {
		params["viewer_lgm.user_id"] = userID
		params["viewer_lgm.role_in_group"] = "student"
	}
	if helpers.IsRole(role, "instructor") {
		params["viewer_lgm.user_id"] = userID
		params["viewer_lgm.role_in_group"] = "instructor"
	}
	if institutionID := ctx.GetString("institution_id"); institutionID != "" && !helpers.IsRole(role, "super_admin") {
		params["i.id"] = institutionID
	}
	viewerJoin := "left join learning_group_members viewer_lgm on viewer_lgm.learning_group_id = lrg.learning_group_id and viewer_lgm.deleted_at is null"
	if !isStudentViewer && !helpers.IsRole(role, "instructor") {
		viewerJoin = "left join learning_group_members viewer_lgm on false"
	}
	baseQuery := `select lr.id, lr.title, lr.description, lr.type, lr.uploaded_user_id,
		viewer_lgm.user_id as viewer_member_user_id, viewer_lgm.role_in_group as viewer_member_role,
		i.id as resource_institution_id,
		lr.created_at, lr.updated_at, lr.deleted_at,
		COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'id', lrf.id,
				'learning_resource_id', lrf.learning_resource_id,
				'type', lrf.type,
				'url', lrf.url,
				'original_name', lrf.original_name,
				'mime_type', lrf.mime_type,
				'size', lrf.size,
				'sort_order', lrf.sort_order,
				'metadata', lrf.metadata,
				'created_at', lrf.created_at,
				'updated_at', lrf.updated_at
			) ORDER BY lrf.sort_order)
			FROM learning_resource_files lrf
			WHERE lrf.learning_resource_id = lr.id
		), '[]'::jsonb) AS files,
		u.name as uploaded_user_name, i.name as institution_name,
		COALESCE(array_agg(lg.id) FILTER (WHERE lg.id IS NOT NULL), '{}') as learning_group_ids,
		COALESCE(string_agg(lg.name, ', ' ORDER BY lg.name) FILTER (WHERE lg.id IS NOT NULL), '') as learning_group_names
		from learning_resources lr
		join users u on u.id = lr.uploaded_user_id
		left join institutions i on i.id = u.institution_id
		left join learning_resource_groups lrg on lrg.learning_resource_id = lr.id
		%s
		left join learning_groups lg on lg.id = lrg.learning_group_id and lg.deleted_at is null`
	baseQuery = fmt.Sprintf(baseQuery, viewerJoin)
	return helpers.BuildPaginatedQuery(ctx, s.DB, params, "learning_resources", baseQuery, "group by lr.id, u.name, i.id, i.name, viewer_lgm.user_id, viewer_lgm.role_in_group", "", dto.SortBy)
}

func (s *LearningResourceService) getByID(ctx *gin.Context, id string) (*models.LearningResource, error) {
	var data models.LearningResource
	err := s.DB.Preload("LearningGroups").Preload("UploadedUser").Preload("ResourceFiles").Where("learning_resources.id = ? AND (learning_resources.uploaded_user_id = ? OR EXISTS (SELECT 1 FROM learning_resource_groups x JOIN learning_group_members m ON m.learning_group_id = x.learning_group_id WHERE x.learning_resource_id = learning_resources.id AND m.user_id = ? AND m.deleted_at IS NULL))", id, ctx.GetString("user_id"), ctx.GetString("user_id")).First(&data).Error
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
	var uploadedURLs []string
	if ctx.Request.MultipartForm != nil && len(ctx.Request.MultipartForm.File["file"]) > 0 {
		var fileErr error
		uploadedURLs, fileErr = uploadLearningResourceFiles(ctx, institutionID)
		if fileErr != nil {
			return nil, fileErr
		}
	}

	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&data).Error; err != nil {
			return err
		}
		if err := syncLearningResourceFiles(tx, data.ID, append(dto.Files, uploadedURLs...)); err != nil {
			return err
		}
		return createResourceGroupLinks(tx, data.ID, groups)
	})
	if err != nil {
		deleteLearningResourceFiles(uploadedURLs)
		return nil, err
	}

	data.LearningGroups = groups
	data.LearningGroupIDs = learningGroupIDs(groups)
	if s.Notifier != nil {
		var recipients []uuid.UUID
		if len(data.LearningGroupIDs) > 0 {
			s.DB.Table("learning_group_members").Where("learning_group_id IN ? AND role_in_group = ? AND deleted_at IS NULL", data.LearningGroupIDs, models.RoleInGroupStudent).Pluck("user_id", &recipients)
		}
		ids := make([]string, 0, len(recipients))
		for _, id := range recipients {
			ids = append(ids, id.String())
		}
		go func() {
			_ = s.Notifier.Publish(context.Background(), notif.Event{Type: notif.EventTypeResourceCreated, Scope: notif.EventScopeUser, Title: "Modul baru tersedia", Message: "Ada modul pembelajaran baru di learning group kamu.", UserID: uploadedUserID.String(), RecipientIDs: ids, EntityID: data.ID.String(), InstitutionID: institutionID, WebURL: "/learning-resources/" + data.ID.String(), MobileRoute: "/learning-resources/" + data.ID.String(), Data: map[string]interface{}{"title": data.Title}, CreatedAt: time.Now()})
		}()
	}
	return &data, nil
}

func (s *LearningResourceService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.LearningResource, error) {
	var data models.LearningResource
	if err := s.DB.Preload("ResourceFiles").Where("id = ? AND uploaded_user_id = ?", id, ctx.GetString("user_id")).First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	fileURLs := make([]string, 0, len(data.ResourceFiles))
	for _, file := range data.ResourceFiles {
		fileURLs = append(fileURLs, file.URL)
	}
	var uploadedURLs []string
	filesChanged := false
	var oldFileIDs []uuid.UUID
	for _, value := range dto.OldFileIDs {
		fileID, parseErr := uuid.Parse(value)
		if parseErr != nil {
			return nil, errors.New("invalid old_file_ids format")
		}
		oldFileIDs = append(oldFileIDs, fileID)
	}
	if len(oldFileIDs) > 0 {
		keep := make(map[uuid.UUID]struct{}, len(oldFileIDs))
		for _, fileID := range oldFileIDs {
			keep[fileID] = struct{}{}
		}
		fileURLs = fileURLs[:0]
		for _, file := range data.ResourceFiles {
			if _, exists := keep[file.ID]; exists {
				fileURLs = append(fileURLs, file.URL)
			}
		}
	}
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
	if dto.Files != nil && len(*dto.Files) > 0 {
		if len(oldFileIDs) == 0 {
			fileURLs = nil
		}
		fileURLs = uniqueFileURLs(append(fileURLs, (*dto.Files)...))
		filesChanged = true
	}

	if ctx.Request.MultipartForm != nil && len(ctx.Request.MultipartForm.File["file"]) > 0 {
		var fileErr error
		uploadedURLs, fileErr = uploadLearningResourceFiles(ctx, institutionID)
		if fileErr != nil {
			return nil, fileErr
		}
		if len(uploadedURLs) > 0 {
			if len(oldFileIDs) == 0 && !filesChanged {
				fileURLs = nil
			}
			fileURLs = uniqueFileURLs(append(fileURLs, uploadedURLs...))
			filesChanged = true
		}
	}
	if len(dto.OldFileIDs) > 0 {
		filesChanged = true
	}

	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&data).Error; err != nil {
			return err
		}

		if dto.LearningGroupIDs == nil {
			if filesChanged {
				return reconcileLearningResourceFiles(tx, data.ID, oldFileIDs, fileURLs)
			}
			return nil
		}

		if err := tx.Where("learning_resource_id = ?", data.ID).
			Delete(&models.LearningResourceGroup{}).Error; err != nil {
			return err
		}
		if err := createResourceGroupLinks(tx, data.ID, groups); err != nil {
			return err
		}
		if filesChanged {
			return reconcileLearningResourceFiles(tx, data.ID, oldFileIDs, fileURLs)
		}
		return nil
	})
	if err != nil {
		deleteLearningResourceFiles(uploadedURLs)
		return nil, err
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

func uniqueFileURLs(urls []string) []string {
	result := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, url := range urls {
		if url == "" {
			continue
		}
		if _, exists := seen[url]; exists {
			continue
		}
		seen[url] = struct{}{}
		result = append(result, url)
	}
	return result
}

func syncLearningResourceFiles(tx *gorm.DB, resourceID uuid.UUID, urls []string) error {
	if err := tx.Where("learning_resource_id = ?", resourceID).Delete(&models.LearningResourceFile{}).Error; err != nil {
		return err
	}
	for index, url := range uniqueFileURLs(urls) {
		fileType := models.LearningResourceFileTypeLink
		lowerURL := strings.ToLower(url)
		if strings.Contains(lowerURL, "youtube.com/") || strings.Contains(lowerURL, "youtu.be/") {
			fileType = models.LearningResourceFileTypeYouTube
		} else if strings.HasPrefix(url, strings.TrimSuffix(stores.MinioPublicURL, "/")) {
			fileType = models.LearningResourceFileTypeFile
		}
		file := models.LearningResourceFile{
			LearningResourceID: resourceID,
			Type:               fileType,
			URL:                url,
			SortOrder:          index,
		}
		if err := tx.Create(&file).Error; err != nil {
			return err
		}
	}
	return nil
}

func reconcileLearningResourceFiles(tx *gorm.DB, resourceID uuid.UUID, keepIDs []uuid.UUID, urls []string) error {
	keep := make(map[uuid.UUID]struct{}, len(keepIDs))
	for _, id := range keepIDs {
		keep[id] = struct{}{}
	}
	var existing []models.LearningResourceFile
	if err := tx.Where("learning_resource_id = ?", resourceID).Find(&existing).Error; err != nil {
		return err
	}
	existingIDs := make(map[uuid.UUID]struct{}, len(existing))
	for _, file := range existing {
		existingIDs[file.ID] = struct{}{}
	}
	for _, fileID := range keepIDs {
		if _, exists := existingIDs[fileID]; !exists {
			return fmt.Errorf("old file %s does not belong to learning resource %s", fileID, resourceID)
		}
	}
	for _, file := range existing {
		if _, ok := keep[file.ID]; !ok {
			if err := tx.Unscoped().Delete(&file).Error; err != nil {
				return err
			}
			deleteLearningResourceFiles([]string{file.URL})
		}
	}
	known := make(map[string]struct{}, len(existing))
	for _, file := range existing {
		if _, ok := keep[file.ID]; ok {
			known[file.URL] = struct{}{}
		}
	}
	for index, url := range uniqueFileURLs(urls) {
		if _, ok := known[url]; ok {
			continue
		}
		fileType := models.LearningResourceFileTypeLink
		lowerURL := strings.ToLower(url)
		if strings.Contains(lowerURL, "youtube.com/") || strings.Contains(lowerURL, "youtu.be/") {
			fileType = models.LearningResourceFileTypeYouTube
		} else if strings.HasPrefix(url, strings.TrimSuffix(stores.MinioPublicURL, "/")) {
			fileType = models.LearningResourceFileTypeFile
		}
		if err := tx.Create(&models.LearningResourceFile{
			LearningResourceID: resourceID, Type: fileType, URL: url, SortOrder: index,
		}).Error; err != nil {
			return err
		}
	}
	return nil
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
	if err := s.DB.Unscoped().Preload("ResourceFiles").First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("learning_resource_id = ?", data.ID).Delete(&models.LearningResourceGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("learning_resource_id = ?", data.ID).Delete(&models.LearningResourceFile{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&data).Error
	}); err != nil {
		return false, err
	}
	fileURLs := make([]string, 0, len(data.ResourceFiles))
	for _, file := range data.ResourceFiles {
		fileURLs = append(fileURLs, file.URL)
	}
	deleteLearningResourceFiles(fileURLs)
	return true, nil
}

func (s *LearningResourceService) removeForUser(ctx *gin.Context, id string, permanent bool) (bool, error) {
	if permanent {
		return s.deleteOwned(ctx, id)
	}
	result := s.DB.Where("id = ? AND uploaded_user_id = ?", id, ctx.GetString("user_id")).Delete(&models.LearningResource{})
	return result.RowsAffected > 0, result.Error
}

func (s *LearningResourceService) deleteOwned(ctx *gin.Context, id string) (bool, error) {
	var data models.LearningResource
	if err := s.DB.Unscoped().Preload("ResourceFiles").Where("id = ? AND uploaded_user_id = ?", id, ctx.GetString("user_id")).First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("learning_resource_id = ?", data.ID).Delete(&models.LearningResourceGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("learning_resource_id = ?", data.ID).Delete(&models.LearningResourceFile{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&data).Error
	}); err != nil {
		return false, err
	}
	for _, file := range data.ResourceFiles {
		deleteLearningResourceFiles([]string{file.URL})
	}
	return true, nil
}
