package auth

import (
	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"
	"clasenna-go-backend/src/users"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	DB       *gorm.DB
	Notifier notif.Publisher
	Events   EventPublisher
}

func NewAuthService(db *gorm.DB, events ...EventPublisher) *AuthService {
	service := &AuthService{DB: db, Notifier: notif.NewPublisherFromEnv()}
	if len(events) > 0 {
		service.Events = events[0]
	}
	return service
}

func (s *AuthService) Register(ctx context.Context, dto RegisterDTO, correlationID string) (*models.User, error) {
	var institution models.Institution
	var data *models.User
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var emailCount int64
		if err := tx.Model(&models.User{}).Where("email = ?", dto.Email).Count(&emailCount).Error; err != nil {
			return err
		}
		if emailCount > 0 {
			return errors.New("email already in use")
		}

		var institutionCount int64
		if err := tx.Model(&models.Institution{}).Where("name = ?", dto.InstitutionName).Count(&institutionCount).Error; err != nil {
			return err
		}
		if institutionCount > 0 {
			return errors.New("institution name already in use")
		}

		institution = models.Institution{Name: dto.InstitutionName, Code: helpers.GenerateInstitutionCode(dto.InstitutionName), Website: dto.Website, Status: models.InstitutionStatusInactive}
		if err := tx.Create(&institution).Error; err != nil {
			return err
		}

		userService := users.UserService{DB: tx}
		createdUser, err := userService.CreateUser(nil, users.CreateDTO{Name: dto.Name, Email: dto.Email, Password: dto.Password, Type: "admin", RoleID: os.Getenv("SEED_ROLE_ID_ADMIN"), InstitutionID: institution.ID.String(), ContextType: "-", ContextCode: "-", IsActive: "active"})
		if err != nil {
			return fmt.Errorf("failed to create institution admin: %w", err)
		}
		if err := tx.Preload("Role").Preload("Institution").First(createdUser, "id = ?", createdUser.ID).Error; err != nil {
			return err
		}
		data = createdUser
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.Events != nil {
		if err := s.Events.Publish(ctx, "auth.register", institution.ID.String(), correlationID, map[string]any{"user_id": data.ID, "email": data.Email}); err != nil {
			slog.ErrorContext(ctx, "failed to publish auth.register", "error", err, "user_id", data.ID)
		}
	}

	return data, nil
}

func (s *AuthService) StudentScanLogin(ctx context.Context, dto StudentScanLoginDTO, correlationID string) (map[string]interface{}, error) {
	var data models.User
	err := s.DB.WithContext(ctx).
		Joins("JOIN institutions ON institutions.id = users.institution_id").
		Where("LOWER(institutions.code) = LOWER(?) AND users.barcode = ? AND (users.type = ? OR users.context_type = ?)", dto.InstitutionCode, dto.Barcode, "student", "student").
		Preload("Role").Preload("Institution").First(&data).Error
	if err != nil || data.PinHash == "" {
		return nil, errors.New("invalid institution, barcode, or pin")
	}
	if data.IsActive != models.UserStatusActive {
		return nil, errors.New("student account is inactive")
	}
	if data.Institution == nil || data.Institution.Status != models.InstitutionStatusActive {
		return nil, errors.New("institution is inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(data.PinHash), []byte(dto.Pin)); err != nil {
		return nil, errors.New("invalid institution, barcode, or pin")
	}

	institutionID := data.InstitutionID.String()
	token, err := cryptography.GenerateTokenWithInstitution(data.ID.String(), data.Email, data.RoleID.String(), institutionID, 12*time.Hour)
	if err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Model(&data).Update("current_token", token).Error; err != nil {
		return nil, err
	}
	if s.Events != nil {
		if err := s.Events.Publish(ctx, "auth.login", institutionID, correlationID, map[string]any{"user_id": data.ID, "method": "student_barcode"}); err != nil {
			slog.ErrorContext(ctx, "failed to publish student auth.login", "error", err, "user_id", data.ID)
		}
	}

	return map[string]interface{}{
		"id": data.ID, "name": data.Name, "role": data.RoleID,
		"avatar_profile_url": data.AvatarURL, "institution": data.Institution,
		"token": token,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, dto LoginDTO, correlationID string) (map[string]interface{}, error) {
	var data models.User
	err := s.DB.Where("email = ?", dto.Email).First(&data).Error
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	var layerOne []int
	err = json.Unmarshal([]byte(data.LayerOne), &layerOne)
	if err != nil {
		return nil, errors.New("invalid encryption layer")
	}

	// Decrypt password from user data
	decrypted := cryptography.PolyalphabetDecrypt(data.Password, data.LayerTwo)
	decrypted = cryptography.VigenereDecrypt(decrypted, layerOne)

	if dto.Password != string(decrypted) {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	var expiration time.Duration
	if dto.RememberMe {
		expiration = 0 // no expiry
	} else {
		expiration = 24 * time.Hour // 1 day
	}

	institutionID := ""
	if data.InstitutionID != nil {
		institutionID = data.InstitutionID.String()
	}
	token, err := cryptography.GenerateTokenWithInstitution(data.ID.String(), data.Email, data.RoleID.String(), institutionID, expiration)
	if err != nil {
		return nil, err
	}

	// Save token to DB
	data.CurrentToken = token
	if err := s.DB.Save(&data).Error; err != nil {
		return nil, err
	}

	if err := s.DB.Preload("Role").Preload("Institution").First(&data, "id = ?", data.ID).Error; err != nil {
		return nil, err
	}

	res := map[string]interface{}{
		"id":                 data.ID,
		"name":               data.Name,
		"email":              data.Email,
		"role":               data.RoleID,
		"avatar_profile_url": data.AvatarURL,
		"institution":        data.Institution,
		"token":              token,
	}
	if s.Events != nil {
		if err := s.Events.Publish(ctx, "auth.login", institutionID, correlationID, map[string]any{"user_id": data.ID, "email": data.Email}); err != nil {
			slog.ErrorContext(ctx, "failed to publish auth.login", "error", err, "user_id", data.ID)
		}
	}

	if s.Notifier != nil {
		roleName := ""
		if data.Role != nil {
			roleName = data.Role.Name
		}

		institutionName := ""
		if data.Institution != nil {
			institutionName = data.Institution.Name
		}

		go func(userID, roleName, institutionName string) {
			_ = s.Notifier.Publish(context.Background(), notif.Event{
				Type:        notif.EventTypeLoginDetected,
				Scope:       notif.EventScopeUser,
				Title:       "Login baru terdeteksi",
				Message:     "Akunmu baru saja masuk dari perangkat baru. Jika itu bukan kamu, segera ubah password.",
				Category:    "Sistem",
				CategoryKey: "system",
				InstitutionID: func() string {
					if data.Institution != nil {
						return data.Institution.ID.String()
					}
					return ""
				}(),
				GroupName: institutionName,
				UserID:    userID,
				ActorID:   userID,
				Data: map[string]interface{}{
					"role": roleName,
				},
				CreatedAt: time.Now(),
			})
		}(data.ID.String(), roleName, institutionName)
	}

	return res, nil
}

func (s *AuthService) NotifyPasswordReset(userID, userName, email string) {
	if s.Notifier == nil {
		return
	}

	go func() {
		_ = s.Notifier.Publish(context.Background(), notif.Event{
			Type:        notif.EventTypePasswordReset,
			Scope:       notif.EventScopeUser,
			Title:       "Password berhasil direset",
			Message:     "Password akunmu baru saja diperbarui. Jika ini bukan kamu, segera ubah kembali.",
			Category:    "Sistem",
			CategoryKey: "system",
			UserID:      userID,
			ActorID:     userID,
			Data: map[string]interface{}{
				"user_name": userName,
				"email":     email,
			},
			CreatedAt: time.Now(),
		})
	}()
}

func (s *AuthService) ResetPassword(ctx context.Context, dto ResetPasswordDTO, correlationID string) error {
	var data models.User
	if err := s.DB.First(&data, "id = ?", dto.UserID).Error; err != nil {
		return errors.New("user not found")
	}

	rand.Seed(time.Now().UnixNano())
	layerOneSize := rand.Intn(6) + 3
	layerTwoSize := rand.Intn(25) + 12
	saltSize := rand.Intn(12) + 8

	layerOne := rand.Perm(layerOneSize)
	layerTwo := cryptography.GenerateRandomString(layerTwoSize)
	salt := cryptography.GenerateRandomString(saltSize)

	encodedOne := cryptography.VigenereEncrypt(dto.Password, layerOne)
	encodedTwo := cryptography.PolyalphabetEncrypt(encodedOne, layerTwo)

	layerOneStr, err := json.Marshal(layerOne)
	if err != nil {
		return err
	}

	data.Password = encodedTwo
	data.LayerOne = string(layerOneStr)
	data.LayerTwo = layerTwo
	data.Salt = salt

	if err := s.DB.Save(&data).Error; err != nil {
		return err
	}

	s.NotifyPasswordReset(data.ID.String(), data.Name, data.Email)
	if s.Events != nil {
		institutionID := ""
		if data.InstitutionID != nil {
			institutionID = data.InstitutionID.String()
		}
		if err := s.Events.Publish(ctx, "auth.password_reset", institutionID, correlationID, map[string]any{"user_id": data.ID}); err != nil {
			slog.ErrorContext(ctx, "failed to publish auth.password_reset", "error", err, "user_id", data.ID)
		}
	}
	return nil
}
