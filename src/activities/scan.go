package activities

import (
	"errors"
	"strings"
	"time"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type scanInput struct {
	ActivityItemID  string
	LearningGroupID string
	Description     string
	Platform        string
	OccurredAt      *time.Time
}

func (s *ActivityService) qrScan(ctx *gin.Context, dto QRScanDTO) (*ScanResult, error) {
	qr, err := helpers.ParseStudentQR(dto.QRCode)
	if err != nil {
		return nil, err
	}
	return s.scan(ctx, scanInput{
		ActivityItemID: dto.ActivityItemID, LearningGroupID: dto.LearningGroupID,
		Description: dto.Description, Platform: dto.Platform, OccurredAt: dto.OccurredAt,
	}, func(q *gorm.DB) *gorm.DB {
		q = q.Where("users.barcode = ?", qr.Barcode)
		if qr.InstitutionCode != "" {
			q = q.Joins("JOIN institutions ON institutions.id = users.institution_id").Where("LOWER(institutions.code) = LOWER(?)", qr.InstitutionCode)
		}
		return q
	}, "active student with this QR code was not found")
}

func (s *ActivityService) rfidScan(ctx *gin.Context, dto RFIDScanDTO) (*ScanResult, error) {
	cardCode := strings.TrimSpace(dto.RFIDCardCode)
	if cardCode == "" {
		return nil, errors.New("rfid_card_code is required")
	}
	return s.scan(ctx, scanInput{
		ActivityItemID: dto.ActivityItemID, LearningGroupID: dto.LearningGroupID,
		Description: dto.Description, Platform: dto.Platform, OccurredAt: dto.OccurredAt,
	}, func(q *gorm.DB) *gorm.DB {
		return q.Where("users.rfid_card_code = ?", cardCode)
	}, "active student with this rfid card was not found")
}

// scan records an activity item for a scanned student (QR code / RFID card), optionally limited to one learning group.
func (s *ActivityService) scan(ctx *gin.Context, dto scanInput, findStudent func(*gorm.DB) *gorm.DB, notFoundMessage string) (*ScanResult, error) {
	recordedUserID, err := uuid.Parse(ctx.GetString("user_id"))
	if err != nil {
		return nil, errors.New("authenticated user_id is required")
	}
	institutionID := ctx.GetString("institution_id")

	itemID, err := uuid.Parse(dto.ActivityItemID)
	if err != nil {
		return nil, errors.New("invalid activity_item_id format")
	}
	item, err := s.findItem(itemID)
	if err != nil {
		return nil, err
	}
	if institutionID != "" && item.InstitutionID != nil && item.InstitutionID.String() != institutionID {
		return nil, errors.New("activity item not found")
	}

	var student models.User
	studentQuery := findStudent(s.DB.Model(&models.User{})).Where("users.type = ? AND users.status = ?", "student", models.UserStatusActive)
	if institutionID != "" {
		studentQuery = studentQuery.Where("users.institution_id = ?", institutionID)
	}
	if err := studentQuery.Select("users.*").First(&student).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(notFoundMessage)
		}
		return nil, err
	}

	if dto.LearningGroupID != "" {
		groupID, err := uuid.Parse(dto.LearningGroupID)
		if err != nil {
			return nil, errors.New("invalid learning_group_id format")
		}
		groupQuery := s.DB.Model(&models.LearningGroup{}).Where("id = ?", groupID)
		if institutionID != "" {
			groupQuery = groupQuery.Where("institution_id = ?", institutionID)
		}
		var groupCount int64
		if err := groupQuery.Count(&groupCount).Error; err != nil {
			return nil, err
		}
		if groupCount == 0 {
			return nil, errors.New("learning group not found")
		}
		var memberCount int64
		if err := s.DB.Model(&models.LearningGroupMember{}).Where("learning_group_id = ? AND user_id = ? AND role_in_group = ? AND deleted_at IS NULL", groupID, student.ID, models.RoleInGroupStudent).Count(&memberCount).Error; err != nil {
			return nil, err
		}
		if memberCount == 0 {
			return nil, errors.New("student is not a member of the learning group")
		}
	}

	occurredAt := time.Now()
	if dto.OccurredAt != nil {
		occurredAt = *dto.OccurredAt
	}
	platform := dto.Platform
	if platform == "" {
		platform = "web"
	}
	data, err := s.create(CreateDTO{
		ActivityItemID: itemID.String(), UserID: student.ID.String(), LearningGroupID: dto.LearningGroupID,
		RecordedUserID: recordedUserID.String(), Description: dto.Description, Platform: platform, OccurredAt: occurredAt,
	})
	if err != nil {
		return nil, err
	}
	return &ScanResult{
		ID: data.ID, UserID: student.ID, Name: student.Name, AvatarProfileURL: student.AvatarURL,
		ActivityItemID: item.ID, ActivityItemName: item.Name, Type: string(item.Type), PointValue: data.PointValue,
		LearningGroupID: data.LearningGroupID, OccurredAt: data.OccurredAt,
	}, nil
}
