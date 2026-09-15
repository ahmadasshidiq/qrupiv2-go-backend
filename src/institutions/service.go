package institutions

import (
	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/seeder"
	"clasenna-go-backend/libs/stores"
	"clasenna-go-backend/src/users"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InstitutionService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *InstitutionService {
	return &InstitutionService{DB: db}
}

func (s *InstitutionService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filters
	params["institutions.deleted_at.isnull"] = ""

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,           // DB
		params,         // filter
		"institutions", // table name
		"",             // optional base query
		"",             // optional query group by
		"",             // optional select fields
		dto.SortBy,     // optional default sort
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *InstitutionService) getByID(id string) (*models.Institution, error) {
	var data models.Institution
	err := s.DB.First(&data, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *InstitutionService) create(ctx *gin.Context, dto CreateDTO) (map[string]interface{}, error) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil && err != http.ErrMissingFile {
		return nil, err
	}
	if file != nil {
		defer file.Close()
	}

	randomPassword := cryptography.GenerateRandomString(10)
	var response map[string]interface{}
	var uploadedURL string

	err = s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var exists int64
		if err := tx.Model(&models.Institution{}).Where("name = ?", dto.Name).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return errors.New("name already in use")
		}
		var subscriptionID *uuid.UUID
		if dto.CurrentSubscriptionID != "" {
			parsedID, err := uuid.Parse(dto.CurrentSubscriptionID)
			if err != nil {
				return errors.New("invalid current_subscription_id format")
			}
			subscriptionID = &parsedID
		}
		region, err := resolveRegionSnapshot(tx, dto.ProvinceCode, dto.RegencyCode, dto.DistrictCode, dto.VillageCode)
		if err != nil {
			return err
		}

		institution := models.Institution{
			CurrentSubscriptionID: subscriptionID,
			Name:                  dto.Name,
			Code:                  helpers.GenerateInstitutionCode(dto.Name),
			AvatarURL:             dto.AvatarURL,
			Address:               dto.Address,
			Latitude:              dto.Latitude,
			Longitude:             dto.Longitude,
			Phone:                 dto.Phone,
			Website:               dto.Website,
			ProvinceCode:          region.ProvinceCode,
			ProvinceName:          region.ProvinceName,
			RegencyCode:           region.RegencyCode,
			RegencyName:           region.RegencyName,
			DistrictCode:          region.DistrictCode,
			DistrictName:          region.DistrictName,
			VillageCode:           region.VillageCode,
			VillageName:           region.VillageName,
			Country:               dto.Country,
			ZipCode:               dto.ZipCode,
			Status:                models.InstitutionStatus(dto.Status),
		}
		if err := tx.Create(&institution).Error; err != nil {
			return err
		}
		if err := createDefaultKAIHActivities(tx, institution.ID); err != nil {
			return fmt.Errorf("failed to create default 7 KAIH activities: %w", err)
		}
		if err := seeder.SeedAttendanceAbsenceReasonsForInstitution(tx, institution.ID); err != nil {
			return fmt.Errorf("failed to create default attendance absence reasons: %w", err)
		}

		if file != nil {
			publicURL, err := stores.UploadToMinio(
				file,
				os.Getenv("MINIO_PRODUCT_BUCKET"),
				institution.ID.String(),
				header.Filename,
				header.Header.Get("Content-Type"),
				header.Size,
			)
			if err != nil {
				return fmt.Errorf("failed to upload avatar: %w", err)
			}
			uploadedURL = publicURL
			institution.AvatarURL = publicURL
			if err := tx.Model(&institution).Update("avatar_url", publicURL).Error; err != nil {
				return err
			}
		}

		userService := users.UserService{DB: tx}
		adminDTO := users.CreateDTO{
			Name:          fmt.Sprintf("Admin %s", dto.Name),
			Email:         dto.Email,
			Password:      randomPassword,
			Type:          "admin",
			RoleID:        os.Getenv("SEED_ROLE_ID_ADMIN"),
			InstitutionID: institution.ID.String(),
			ContextType:   "-",
			ContextCode:   "-",
			Status:        "active",
		}
		user, err := userService.CreateUser(nil, adminDTO)
		if err != nil {
			return fmt.Errorf("failed to create institution admin: %w", err)
		}

		response = map[string]interface{}{
			"id":       institution.ID,
			"email":    user.Email,
			"password": randomPassword,
		}
		return nil
	})
	if err != nil {
		if uploadedURL != "" {
			stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), uploadedURL)
		}
		return nil, err
	}

	return response, nil
}

func createDefaultKAIHActivities(tx *gorm.DB, institutionID uuid.UUID) error {
	category, items := defaultKAIHActivities(institutionID)
	if err := tx.Create(&category).Error; err != nil {
		return err
	}
	for index := range items {
		items[index].CategoryID = &category.ID
	}
	return tx.Create(&items).Error
}

func defaultKAIHActivities(institutionID uuid.UUID) (models.ActivityCategory, []models.ActivityItem) {
	category := models.ActivityCategory{
		InstitutionID: &institutionID,
		Name:          "7 KAIH",
		Description:   "7 Kebiasaan Anak Indonesia Hebat",
	}

	names := []string{
		"Bangun Pagi",
		"Beribadah",
		"Bermasyarakat",
		"Berolahraga",
		"Gemar Belajar",
		"Makan Sehat dan Bergizi",
		"Tidur Cepat",
	}
	items := make([]models.ActivityItem, len(names))
	for index, name := range names {
		items[index] = models.ActivityItem{
			InstitutionID: &institutionID,
			Name:          name,
			Type:          models.ActivityItemTypePositive,
			PointValue:    1,
			DailyLimit:    1,
			PeriodLimit:   0,
			PeriodType:    models.ActivityLimitNone,
		}
	}
	return category, items
}

func (s *InstitutionService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.Institution, error) {
	var data models.Institution
	if err := s.DB.First(&data, "id = ?", id).Error; err != nil {
		return nil, errors.New("institution not found")
	}
	oldAvatarURL := data.AvatarURL

	if dto.Name != nil {
		data.Name = *dto.Name
	}
	if dto.AvatarURL != nil {
		data.AvatarURL = *dto.AvatarURL
	}

	if dto.CurrentSubscriptionID != nil {
		id, err := uuid.Parse(*dto.CurrentSubscriptionID)
		if err != nil {
			return nil, errors.New("invalid current_subscription_id format")
		}
		data.CurrentSubscriptionID = &id
	}

	if dto.Address != nil {
		data.Address = *dto.Address
	}

	if dto.Latitude != nil {
		data.Latitude = *dto.Latitude
	}

	if dto.Longitude != nil {
		data.Longitude = *dto.Longitude
	}
	if dto.Phone != nil {
		data.Phone = *dto.Phone
	}
	if dto.Website != nil {
		data.Website = *dto.Website
	}
	if dto.Country != nil {
		data.Country = *dto.Country
	}
	if dto.ProvinceCode != nil || dto.RegencyCode != nil || dto.DistrictCode != nil || dto.VillageCode != nil {
		provinceCode := data.ProvinceCode
		regencyCode := data.RegencyCode
		districtCode := data.DistrictCode
		villageCode := data.VillageCode
		if dto.ProvinceCode != nil {
			provinceCode = *dto.ProvinceCode
		}
		if dto.RegencyCode != nil {
			regencyCode = *dto.RegencyCode
		}
		if dto.DistrictCode != nil {
			districtCode = *dto.DistrictCode
		}
		if dto.VillageCode != nil {
			villageCode = *dto.VillageCode
		}
		region, err := resolveRegionSnapshot(s.DB, provinceCode, regencyCode, districtCode, villageCode)
		if err != nil {
			return nil, err
		}
		data.ProvinceCode = region.ProvinceCode
		data.ProvinceName = region.ProvinceName
		data.RegencyCode = region.RegencyCode
		data.RegencyName = region.RegencyName
		data.DistrictCode = region.DistrictCode
		data.DistrictName = region.DistrictName
		data.VillageCode = region.VillageCode
		data.VillageName = region.VillageName
	}
	if dto.ZipCode != nil {
		data.ZipCode = *dto.ZipCode
	}

	if dto.Status != nil {
		data.Status = models.InstitutionStatus(*dto.Status)
	}

	var uploadedURL string
	file, header, err := ctx.Request.FormFile("file")
	if err == nil {
		defer file.Close()

		publicURL, err := stores.UploadToMinio(
			file,
			os.Getenv("MINIO_PRODUCT_BUCKET"),
			id,
			header.Filename,
			header.Header.Get("Content-Type"),
			header.Size,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upload avatar: %w", err)
		}

		uploadedURL = publicURL
		data.AvatarURL = publicURL
	} else if err != http.ErrMissingFile {
		return nil, err
	}

	if err := s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		return tx.Save(&data).Error
	}); err != nil {
		if uploadedURL != "" {
			stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), uploadedURL)
		}
		return nil, err
	}
	if uploadedURL != "" && oldAvatarURL != "" && oldAvatarURL != uploadedURL {
		stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), oldAvatarURL)
	}

	return &data, nil
}

type regionSnapshot struct {
	ProvinceCode string
	ProvinceName string
	RegencyCode  string
	RegencyName  string
	DistrictCode string
	DistrictName string
	VillageCode  string
	VillageName  string
}

func resolveRegionSnapshot(db *gorm.DB, provinceCode, regencyCode, districtCode, villageCode string) (regionSnapshot, error) {
	result := regionSnapshot{}
	if provinceCode == "" && regencyCode == "" && districtCode == "" && villageCode == "" {
		return result, nil
	}
	if provinceCode == "" {
		return result, errors.New("province_code is required when regional data is provided")
	}

	var province models.Province
	if err := db.Where("code = ? AND is_active = true", provinceCode).First(&province).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("province not found")
		}
		return result, err
	}
	result.ProvinceCode, result.ProvinceName = province.Code, province.Name
	if regencyCode == "" {
		if districtCode != "" || villageCode != "" {
			return result, errors.New("regency_code is required when district or village is provided")
		}
		return result, nil
	}

	var regency models.Regency
	if err := db.Where("code = ? AND province_code = ? AND is_active = true", regencyCode, provinceCode).First(&regency).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("regency not found in selected province")
		}
		return result, err
	}
	result.RegencyCode, result.RegencyName = regency.Code, regency.Name
	if districtCode == "" {
		if villageCode != "" {
			return result, errors.New("district_code is required when village is provided")
		}
		return result, nil
	}

	var district models.District
	if err := db.Where("code = ? AND regency_code = ? AND is_active = true", districtCode, regencyCode).First(&district).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("district not found in selected regency")
		}
		return result, err
	}
	result.DistrictCode, result.DistrictName = district.Code, district.Name
	if villageCode == "" {
		return result, nil
	}

	var village models.Village
	if err := db.Where("code = ? AND district_code = ? AND is_active = true", villageCode, districtCode).First(&village).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, errors.New("village not found in selected district")
		}
		return result, err
	}
	result.VillageCode, result.VillageName = village.Code, village.Name
	return result, nil
}

func (s *InstitutionService) archive(id string) (bool, error) {
	var data models.Institution
	if err := s.DB.First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := s.DB.Model(&data).Update("deleted_at", time.Now()).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (s *InstitutionService) delete(id string) (bool, error) {
	var data models.Institution
	if err := s.DB.Unscoped().First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Institution creation also creates an administrator. Remove every user
		// owned by the institution first because the users foreign key prevents
		// the parent row from being permanently deleted.
		if err := tx.Unscoped().Where("institution_id = ?", data.ID).Delete(&models.User{}).Error; err != nil {
			return fmt.Errorf("failed to delete institution users: %w", err)
		}
		if err := tx.Unscoped().Delete(&data).Error; err != nil {
			return fmt.Errorf("failed to delete institution: %w", err)
		}
		return nil
	}); err != nil {
		return false, err
	}
	if data.AvatarURL != "" {
		stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), data.AvatarURL)
	}
	return true, nil
}
