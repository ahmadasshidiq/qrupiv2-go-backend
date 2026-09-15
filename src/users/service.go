package users

import (
	"bytes"
	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"
	"clasenna-go-backend/libs/stores"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	DB       *gorm.DB
	Notifier notif.Publisher
	Events   EventPublisher
}

func NewService(db *gorm.DB, publishers ...EventPublisher) *UserService {
	service := &UserService{DB: db, Notifier: notif.NewPublisherFromEnv()}
	if len(publishers) > 0 {
		service.Events = publishers[0]
	}
	return service
}

// for public func
func (s *UserService) CreateUser(ctx *gin.Context, dto CreateDTO) (*models.User, error) {
	return s.create(ctx, dto)
}

func (s *UserService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filter
	params["u.deleted_at.isnull"] = ""
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		params["u.institution_id"] = institutionID
	}

	// default base query
	baseQuery := `
		select 
			u.id,
			u.email,
			u.name,
			u.context_type,
			u.context_code,
			u.type,
			u.phone,
			u.barcode,
			u.status,
			u.avatar_url,
			u.role_id,
			r.name as role_name,
			u.institution_id,
			i.name as institution_name,
			u.created_at
		from users u
		join roles r on r.id = u.role_id
		left join institutions i on i.id = u.institution_id
	`

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,       // DB
		params,     // filter
		"users",    // table name
		baseQuery,  // optional base query
		"",         // optional query group by
		"",         // optional select fields
		dto.SortBy, // optional default sort
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *UserService) getByID(ctx *gin.Context, id string) (*models.User, error) {
	var data models.User
	query := s.DB.Where("id = ?", id)
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		query = query.Where("institution_id = ?", institutionID)
	}
	err := query.First(&data).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *UserService) create(ctx *gin.Context, dto CreateDTO) (*models.User, error) {
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		if dto.InstitutionID != "" && dto.InstitutionID != institutionID {
			return nil, errors.New("cannot create user for another institution")
		}
		dto.InstitutionID = institutionID
	}
	userType := dto.Type
	if userType == "" {
		if dto.ContextType == "student" {
			userType = "student"
		} else {
			userType = "staff"
		}
	}
	if userType == "student" {
		if dto.Pin == "" {
			return nil, errors.New("pin is required for student")
		}
		if dto.Email == "" {
			dto.Email = "student-" + uuid.NewString() + "@internal.qrupi"
		}
		if dto.Password == "" {
			dto.Password = cryptography.GenerateRandomString(32)
		}
	} else if dto.Email == "" || dto.Password == "" {
		return nil, errors.New("email and password are required for non-student user")
	}

	var exists int64
	s.DB.Model(&models.User{}).Where("email = ?", dto.Email).Count(&exists)
	if exists > 0 {
		return nil, errors.New("email already in use")
	}

	// Generate salt
	rand.Seed(time.Now().UnixNano())
	layerOneSize := rand.Intn(6) + 3
	layerTwoSize := rand.Intn(25) + 12
	saltSize := rand.Intn(12) + 8

	layerOne := rand.Perm(layerOneSize)
	layerTwo := cryptography.GenerateRandomString(layerTwoSize)
	salt := cryptography.GenerateRandomString(saltSize)

	// Enkripsi password
	encodedOne := cryptography.VigenereEncrypt(dto.Password, layerOne)
	encodedTwo := cryptography.PolyalphabetEncrypt(encodedOne, layerTwo)

	// Simpan layerOne sebagai string JSON
	layerOneStr, err := json.Marshal(layerOne)
	if err != nil {
		return nil, err
	}

	roleID, err := uuid.Parse(dto.RoleID)
	if err != nil {
		return nil, errors.New("invalid role_id format")
	}

	var instID *uuid.UUID
	if dto.InstitutionID != "" {
		id, err := uuid.Parse(dto.InstitutionID)
		if err != nil {
			return nil, errors.New("invalid institution_id format")
		}
		instID = &id
	}

	var barcodeValue, pinHash string
	if userType == "student" {
		if instID == nil {
			return nil, errors.New("institution_id is required for student")
		}
		barcodeValue, err = s.generateUniqueBarcode()
		if err != nil {
			return nil, err
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(dto.Pin), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("hash student pin: %w", hashErr)
		}
		pinHash = string(hash)
	}

	data := models.User{
		Name:          dto.Name,
		Email:         dto.Email,
		RoleID:        roleID,
		InstitutionID: instID,
		Type:          userType,
		Phone:         dto.Phone,
		ContextType:   dto.ContextType,
		ContextCode:   dto.ContextCode,
		Status:        models.UserStatus(dto.Status),
		Password:      encodedTwo,
		LayerOne:      string(layerOneStr),
		LayerTwo:      layerTwo,
		Salt:          salt,
		PinHash:       pinHash,
		Barcode:       barcodeValue,
		AvatarURL:     dto.AvatarURL,
	}

	var uploadedURL string
	if ctx != nil {
		file, header, fileErr := ctx.Request.FormFile("file")
		if fileErr == nil {
			defer file.Close()
			var subFolder string
			if dto.InstitutionID != "" {
				subFolder = dto.InstitutionID
			} else {
				subFolder = "public"
			}

			publicURL, uploadErr := stores.UploadToMinio(
				file,
				os.Getenv("MINIO_PRODUCT_BUCKET"),
				subFolder,
				header.Filename,
				header.Header.Get("Content-Type"),
				header.Size,
			)
			if uploadErr != nil {
				return nil, fmt.Errorf("failed to upload avatar: %w", uploadErr)
			}
			uploadedURL = publicURL
			data.AvatarURL = publicURL
		} else if fileErr != http.ErrMissingFile {
			return nil, fileErr
		}
	}

	dbCtx := context.Background()
	if ctx != nil {
		dbCtx = ctx.Request.Context()
	}
	if err := s.DB.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&data).Error
	}); err != nil {
		if uploadedURL != "" {
			stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), uploadedURL)
		}
		return nil, err
	}

	// Side effects are emitted only after the database commit succeeds.
	if s.Events != nil {
		correlationID := ""
		if ctx != nil {
			correlationID = ctx.GetString("correlation_id")
		}
		if err := s.Events.UserCreated(context.Background(), data, userType == "student", correlationID); err != nil {
			slog.Error("failed to publish user created event", "error", err, "user_id", data.ID)
		}
	}
	if s.Notifier != nil {
		go func(userID, userName, email string) {
			_ = s.Notifier.Publish(context.Background(), notif.Event{
				Type:        notif.EventTypeUserCreated,
				Scope:       notif.EventScopeUser,
				Title:       "User baru ditambahkan",
				Message:     "Akun user baru sudah berhasil dibuat dan siap digunakan.",
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
		}(data.ID.String(), data.Name, data.Email)
	}
	return &data, nil
}

func (s *UserService) generateUniqueBarcode() (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		random := make([]byte, 12)
		if _, err := cryptorand.Read(random); err != nil {
			return "", fmt.Errorf("generate barcode: %w", err)
		}
		value := "STU-" + strings.ToUpper(hex.EncodeToString(random))
		var count int64
		if err := s.DB.Model(&models.User{}).Where("barcode = ?", value).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return value, nil
		}
	}
	return "", errors.New("unable to generate unique barcode")
}

func (s *UserService) getQRCode(ctx *gin.Context, id string) (string, error) {
	var user models.User
	query := s.DB.Select("users.id", "users.type", "users.barcode", "users.institution_id").Joins("JOIN institutions ON institutions.id = users.institution_id").Where("users.id = ?", id)
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		query = query.Where("institution_id = ?", institutionID)
	}
	if err := query.First(&user).Error; err != nil {
		return "", errors.New("user not found")
	}
	if user.Type != "student" || user.Barcode == "" {
		return "", errors.New("student QR code not found")
	}
	var institution models.Institution
	if err := s.DB.Select("code").First(&institution, "id = ?", user.InstitutionID).Error; err != nil {
		return "", err
	}
	data, err := json.Marshal(map[string]string{"institution_code": institution.Code, "barcode": user.Barcode})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *UserService) NotifyPasswordReset(userID, userName string) {
	if s.Notifier == nil {
		return
	}

	go func() {
		_ = s.Notifier.Publish(context.Background(), notif.Event{
			Type:        notif.EventTypePasswordReset,
			Scope:       notif.EventScopeUser,
			Title:       "Password direset",
			Message:     "Password akunmu telah diperbarui. Jika ini bukan kamu, segera hubungi admin.",
			Category:    "Sistem",
			CategoryKey: "system",
			UserID:      userID,
			ActorID:     userID,
			Data: map[string]interface{}{
				"user_name": userName,
			},
			CreatedAt: time.Now(),
		})
	}()
}

func (s *UserService) update(ctx *gin.Context, id string, dto UpdateDTO) (*models.User, error) {
	var data models.User
	query := s.DB.Where("id = ?", id)
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		query = query.Where("institution_id = ?", institutionID)
		if dto.InstitutionID != nil && *dto.InstitutionID != institutionID {
			return nil, errors.New("cannot move user to another institution")
		}
	}
	if err := query.First(&data).Error; err != nil {
		return nil, errors.New("user not found")
	}
	oldAvatarURL := data.AvatarURL

	// Check email if found, err message already use
	if dto.Email != nil && *dto.Email != data.Email {
		var exists int64
		s.DB.Model(&models.User{}).Where("email = ? and id != ?", dto.Email, id).Count(&exists)
		if exists > 0 {
			return nil, errors.New("email already in use")
		}
		data.Email = *dto.Email
	}

	if dto.Name != nil {
		data.Name = *dto.Name
	}

	if dto.RoleID != nil {
		id, err := uuid.Parse(*dto.RoleID)
		if err != nil {
			return nil, errors.New("invalid role_id format")
		}

		data.RoleID = id
	}

	if dto.InstitutionID != nil {
		id, err := uuid.Parse(*dto.InstitutionID)
		if err != nil {
			return nil, errors.New("invalid institution_id format")
		}

		data.InstitutionID = &id
	}

	if dto.ContextType != nil {
		data.ContextType = *dto.ContextType
	}
	if dto.Phone != nil {
		data.Phone = *dto.Phone
	}
	if dto.AvatarURL != nil {
		data.AvatarURL = *dto.AvatarURL
	}
	if dto.Type != nil {
		data.Type = *dto.Type
	}
	if dto.Pin != nil {
		if data.Type != "student" {
			return nil, errors.New("pin can only be set for student")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*dto.Pin), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash student pin: %w", err)
		}
		data.PinHash = string(hash)
	}

	if dto.ContextCode != nil {
		data.ContextCode = *dto.ContextCode
	}

	if dto.Status != nil {
		data.Status = models.UserStatus(*dto.Status)
	}

	if dto.Password != nil {
		// Re-generate encryption layers
		rand.Seed(time.Now().UnixNano())
		layerOneSize := rand.Intn(6) + 3
		layerTwoSize := rand.Intn(25) + 12
		saltSize := rand.Intn(12) + 8

		layerOne := rand.Perm(layerOneSize)
		layerTwo := cryptography.GenerateRandomString(layerTwoSize)
		salt := cryptography.GenerateRandomString(saltSize)

		encodedOne := cryptography.VigenereEncrypt(*dto.Password, layerOne)
		encodedTwo := cryptography.PolyalphabetEncrypt(encodedOne, layerTwo)

		layerOneStr, err := json.Marshal(layerOne)
		if err != nil {
			return nil, err
		}

		data.Password = encodedTwo
		data.LayerOne = string(layerOneStr)
		data.LayerTwo = layerTwo
		data.Salt = salt
	}

	var uploadedURL string
	file, header, err := ctx.Request.FormFile("file")
	if err == nil {
		defer file.Close()

		var subFolder string
		if data.InstitutionID != nil {
			subFolder = data.InstitutionID.String()
		} else {
			subFolder = "public"
		}

		publicURL, err := stores.UploadToMinio(
			file,
			os.Getenv("MINIO_PRODUCT_BUCKET"),
			subFolder,
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

func (s *UserService) archive(ctx *gin.Context, id string) (bool, error) {
	var data models.User
	query := s.DB.Where("id = ?", id)
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		query = query.Where("institution_id = ?", institutionID)
	}
	if err := query.First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := s.DB.Model(&data).Update("deleted_at", time.Now()).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (s *UserService) delete(ctx *gin.Context, id string) (bool, error) {
	var data models.User
	query := s.DB.Unscoped().Where("id = ?", id)
	if institutionID, scoped := authenticatedInstitutionID(ctx); scoped {
		query = query.Where("institution_id = ?", institutionID)
	}
	if err := query.First(&data).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := s.DB.Unscoped().Where("id = ?", id).Delete(&models.User{}).Error; err != nil {
		return false, err
	}
	if data.AvatarURL != "" {
		stores.DeleteMinioFiles(os.Getenv("MINIO_PRODUCT_BUCKET"), data.AvatarURL)
	}
	return true, nil
}

func (s *UserService) generateTemplateExcel() ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheet := "Users"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Name",
		"Email",
		"Password",
		"Role Name",
		"Institution Code",
		"Phone",
		"PIN (Student)",
		"Context Type",
		"Context Code",
		"Status",
	}

	headerComments := map[string]string{
		"A1": "Isi nama lengkap user. Contoh: Ahmad Fauzi",
		"B1": "Isi email aktif dengan format valid. Contoh: ahmad@gmail.com",
		"C1": "Isi password awal user. Contoh: Password123",
		"D1": "Pilih salah satu: Admin, Instructor, Student",
		"E1": "Isi kode institusi. Kode dapat dilihat pada Dashboard.",
		"F1": "Isi nomor telepon user.",
		"G1": "Wajib untuk siswa. Isi PIN numerik 4 sampai 8 digit.",
		"H1": "Isi jenis identitas/konteks sesuai kebutuhan di lapangan. Contoh: NIP, NIK, NIM, Employee ID, Member ID, Vendor Code, atau kode lain yang digunakan institusi.",
		"I1": "Isi kode sesuai Context Type. Contoh: jika Context Type NIP, isi nomor NIP. Contoh: 0477093",
		"J1": "Pilih salah satu: Active atau Inactive",
	}

	// Style header
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"3733ab"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D9E1F2", Style: 1},
			{Type: "right", Color: "D9E1F2", Style: 1},
			{Type: "top", Color: "D9E1F2", Style: 1},
			{Type: "bottom", Color: "D9E1F2", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Tulis header + comment
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, err
		}

		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}

		if err := f.SetCellStyle(sheet, cell, cell, headerStyle); err != nil {
			return nil, err
		}

		if comment, ok := headerComments[cell]; ok {
			if err := f.AddComment(sheet, excelize.Comment{
				Cell:   cell,
				Author: "System",
				Text:   comment,
				Width:  260,
				Height: 90,
			}); err != nil {
				return nil, err
			}
		}
	}

	// Lebar kolom
	colWidths := map[string]float64{
		"A": 24,
		"B": 28,
		"C": 20,
		"D": 18,
		"E": 22,
		"F": 18,
		"G": 22,
		"H": 20,
		"I": 18,
		"J": 18,
	}

	for col, width := range colWidths {
		if err := f.SetColWidth(sheet, col, col, width); err != nil {
			return nil, err
		}
	}

	// Range input user
	inputRangeStart := 2
	inputRangeEnd := 1000

	// Dropdown Role Name: Admin, Instructor, Student
	if err := addDropdownValidation(
		f,
		sheet,
		fmt.Sprintf("D%d:D%d", inputRangeStart, inputRangeEnd),
		[]string{"Admin", "Instructor", "Student"},
		"Role Name",
		"Pilih salah satu: Admin, Instructor, Student",
	); err != nil {
		return nil, err
	}

	// Dropdown Context Type: NIP, NIK, NIM
	if err := addDropdownValidation(
		f,
		sheet,
		fmt.Sprintf("H%d:H%d", inputRangeStart, inputRangeEnd),
		[]string{"NIP", "NIK", "NIM"},
		"Context Type",
		"Pilih salah satu: NIP, NIK, NIM",
	); err != nil {
		return nil, err
	}

	// Dropdown Status: Active, In Active, Graduated
	if err := addDropdownValidation(
		f,
		sheet,
		fmt.Sprintf("J%d:J%d", inputRangeStart, inputRangeEnd),
		[]string{"active", "inactive"},
		"Status",
		"Pilih salah satu: active, inactive",
	); err != nil {
		return nil, err
	}

	// Style conditional merah untuk input tidak valid
	invalidStyle, err := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "9C0006",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFC7CE"},
			Pattern: 1,
		},
	})
	if err != nil {
		return nil, err
	}

	// Email invalid: tidak kosong tapi tidak mengandung @ atau titik
	// Contoh "ahmad" akan merah.
	if err := f.SetConditionalFormat(
		sheet,
		fmt.Sprintf("B%d:B%d", inputRangeStart, inputRangeEnd),
		[]excelize.ConditionalFormatOptions{
			{
				Type:     "formula",
				Criteria: `=AND(B2<>"",OR(ISERROR(SEARCH("@",B2)),ISERROR(SEARCH(".",B2))))`,
				Format:   &invalidStyle,
			},
		},
	); err != nil {
		return nil, err
	}

	// Role Name invalid jika bukan Admin, Instructor, Student
	if err := f.SetConditionalFormat(
		sheet,
		fmt.Sprintf("D%d:D%d", inputRangeStart, inputRangeEnd),
		[]excelize.ConditionalFormatOptions{
			{
				Type:     "formula",
				Criteria: `=AND(D2<>"",ISERROR(MATCH(D2,{"Admin","Instructor","Student"},0)))`,
				Format:   &invalidStyle,
			},
		},
	); err != nil {
		return nil, err
	}

	// Context Type invalid jika bukan NIP, NIK, NIM
	if err := f.SetConditionalFormat(
		sheet,
		fmt.Sprintf("H%d:H%d", inputRangeStart, inputRangeEnd),
		[]excelize.ConditionalFormatOptions{
			{
				Type:     "formula",
				Criteria: `=AND(H2<>"",ISERROR(MATCH(H2,{"NIP","NIK","NIM"},0)))`,
				Format:   &invalidStyle,
			},
		},
	); err != nil {
		return nil, err
	}

	// PIN wajib 4-8 digit untuk Student.
	if err := f.SetConditionalFormat(
		sheet,
		fmt.Sprintf("G%d:G%d", inputRangeStart, inputRangeEnd),
		[]excelize.ConditionalFormatOptions{{
			Type:     "formula",
			Criteria: `=AND(D2="Student",OR(LEN(G2)<4,LEN(G2)>8,NOT(ISNUMBER(--G2))))`,
			Format:   &invalidStyle,
		}},
	); err != nil {
		return nil, err
	}

	// Status invalid jika bukan active atau inactive.
	if err := f.SetConditionalFormat(
		sheet,
		fmt.Sprintf("J%d:J%d", inputRangeStart, inputRangeEnd),
		[]excelize.ConditionalFormatOptions{
			{
				Type:     "formula",
				Criteria: `=AND(J2<>"",ISERROR(MATCH(J2,{"active","inactive"},0)))`,
				Format:   &invalidStyle,
			},
		},
	); err != nil {
		return nil, err
	}

	// Tambah sheet panduan
	if err := addGuideSheet(f); err != nil {
		return nil, err
	}

	// Freeze header
	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return nil, err
	}

	// Auto filter
	if err := f.AutoFilter(sheet, "A1:J1", nil); err != nil {
		return nil, err
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func addDropdownValidation(
	f *excelize.File,
	sheet string,
	cellRange string,
	values []string,
	title string,
	message string,
) error {
	dv := excelize.NewDataValidation(true)
	dv.SetSqref(cellRange)

	if err := dv.SetDropList(values); err != nil {
		return err
	}

	dv.SetInput(title, message)
	dv.SetError(
		excelize.DataValidationErrorStyleStop,
		"Input tidak valid",
		message,
	)

	return f.AddDataValidation(sheet, dv)
}

func addGuideSheet(f *excelize.File) error {
	guideSheet := "Guide"

	if _, err := f.NewSheet(guideSheet); err != nil {
		return err
	}

	rows := [][]interface{}{
		{"Column", "Cara Pengisian", "Contoh"},
		{"Name", "Isi nama lengkap user.", "Ahmad Fauzi"},
		{"Email", "Isi email aktif dengan format email yang valid.", "ahmad@gmail.com"},
		{"Password", "Isi password awal user.", "Password123"},
		{"Role Name", "Pilih salah satu: Admin, Instructor, Student.", "Admin"},
		{"Institution Code", "Isi kode institusi. Kode dapat dilihat pada Dashboard.", "INST001"},
		{"Phone", "Isi nomor telepon user.", "081234567890"},
		{"PIN (Student)", "Wajib untuk siswa, berupa 4 sampai 8 digit angka.", "123456"},
		{"Context Type", "Pilih salah satu: NIP, NIK, NIM.", "NIP"},
		{"Context Code", "Isi kode yang berhubungan dengan Context Type.", "0477093"},
		{"Status", "Pilih salah satu: Active atau Inactive.", "Active"},
	}

	for i, row := range rows {
		cell := fmt.Sprintf("A%d", i+1)
		if err := f.SetSheetRow(guideSheet, cell, &row); err != nil {
			return err
		}
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"70AD47"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	if err := f.SetCellStyle(guideSheet, "A1", "C1", headerStyle); err != nil {
		return err
	}

	if err := f.SetColWidth(guideSheet, "A", "A", 22); err != nil {
		return err
	}

	if err := f.SetColWidth(guideSheet, "B", "B", 55); err != nil {
		return err
	}

	if err := f.SetColWidth(guideSheet, "C", "C", 25); err != nil {
		return err
	}

	return nil
}

func (s *UserService) importUsersByExcel(ctx *gin.Context, file io.Reader) (int, []string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, nil, err
	}

	rows, err := f.GetRows("Users")
	if err != nil {
		return 0, nil, err
	}
	if len(rows) < 2 {
		return 0, nil, errors.New("excel file has no data rows")
	}

	var inserted int
	var rowErrors []string

	for i := 1; i < len(rows); i++ {
		r := rows[i]
		get := func(idx int) string {
			if idx >= len(r) {
				return ""
			}
			return strings.TrimSpace(r[idx])
		}

		name := get(0)
		email := get(1)
		password := get(2)
		roleName := get(3)
		institutionCode := get(4)
		phone := get(5)
		pin := get(6)
		contextType := get(7)
		contextCode := get(8)
		isActive := strings.ToLower(get(9))

		if name == "" || email == "" || password == "" || roleName == "" || isActive == "" {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: name,email,password,role_name,status wajib diisi", i+1))
			continue
		}
		if isActive != string(models.UserStatusActive) && isActive != string(models.UserStatusInactive) {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: status harus active atau inactive", i+1))
			continue
		}

		var role models.Role
		if err := s.DB.Where("LOWER(name) = LOWER(?)", roleName).First(&role).Error; err != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: role '%s' tidak ditemukan", i+1, roleName))
			continue
		}

		var instID *uuid.UUID
		if authenticatedID, scoped := authenticatedInstitutionID(ctx); scoped {
			parsedID, err := uuid.Parse(authenticatedID)
			if err != nil {
				return inserted, rowErrors, errors.New("invalid authenticated institution")
			}
			instID = &parsedID
		} else if institutionCode != "" {
			var institution models.Institution
			if err := s.DB.Where("LOWER(code) = LOWER(?)", institutionCode).First(&institution).Error; err != nil {
				rowErrors = append(rowErrors, fmt.Sprintf("row %d: institution code '%s' tidak ditemukan", i+1, institutionCode))
				continue
			}
			instID = &institution.ID
		}

		var exists int64
		s.DB.Model(&models.User{}).Where("email = ?", email).Count(&exists)
		if exists > 0 {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: email '%s' sudah terpakai", i+1, email))
			continue
		}

		rand.Seed(time.Now().UnixNano())
		layerOne := rand.Perm(rand.Intn(6) + 3)
		layerTwo := cryptography.GenerateRandomString(rand.Intn(25) + 12)
		salt := cryptography.GenerateRandomString(rand.Intn(12) + 8)
		encodedOne := cryptography.VigenereEncrypt(password, layerOne)
		encodedTwo := cryptography.PolyalphabetEncrypt(encodedOne, layerTwo)
		layerOneStr, err := json.Marshal(layerOne)
		if err != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: gagal memproses password", i+1))
			continue
		}

		userType := userTypeFromRoleName(roleName)
		var barcodeValue, pinHash string
		if userType == "student" {
			if len(pin) < 4 || len(pin) > 8 || strings.Trim(pin, "0123456789") != "" {
				rowErrors = append(rowErrors, fmt.Sprintf("row %d: PIN siswa harus 4 sampai 8 digit angka", i+1))
				continue
			}
			barcodeValue, err = s.generateUniqueBarcode()
			if err != nil {
				rowErrors = append(rowErrors, fmt.Sprintf("row %d: gagal membuat barcode", i+1))
				continue
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
			if err != nil {
				rowErrors = append(rowErrors, fmt.Sprintf("row %d: gagal memproses PIN", i+1))
				continue
			}
			pinHash = string(hash)
		}

		data := models.User{
			Name:          name,
			Email:         email,
			RoleID:        role.ID,
			InstitutionID: instID,
			Type:          userType,
			Phone:         phone,
			ContextType:   contextType,
			ContextCode:   contextCode,
			Status:        models.UserStatus(isActive),
			Password:      encodedTwo,
			LayerOne:      string(layerOneStr),
			LayerTwo:      layerTwo,
			Salt:          salt,
			PinHash:       pinHash,
			Barcode:       barcodeValue,
		}

		if err := s.DB.Create(&data).Error; err != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: %s", i+1, err.Error()))
			continue
		}
		inserted++
	}

	return inserted, rowErrors, nil
}

func (s *UserService) importUsersByExcelFromBytes(b []byte) (int, []string, error) {
	return s.importUsersByExcel(nil, bytes.NewReader(b))
}

func authenticatedInstitutionID(ctx *gin.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	value, exists := ctx.Get("institution_id")
	if !exists {
		return "", false
	}
	institutionID, ok := value.(string)
	return institutionID, ok && institutionID != ""
}

func userTypeFromRoleName(roleName string) string {
	switch strings.ToLower(strings.TrimSpace(roleName)) {
	case "student", "siswa":
		return "student"
	case "teacher", "instructor", "guru":
		return "teacher"
	case "admin", "super admin":
		return "admin"
	default:
		return "staff"
	}
}
