package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserImportJob struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey" json:"job_id"`
	UploaderID    string        `gorm:"type:uuid;index" json:"uploader_id"`
	InstitutionID string        `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	Status        string        `gorm:"type:varchar(20);index" json:"status"`
	TotalData     int           `json:"total_data"`
	ProcessedData int           `json:"processed_data"`
	FailedData    int           `json:"failed_data"`
	Errors        datatypesJSON `gorm:"type:jsonb" json:"errors,omitempty"`
	ErrorMessage  string        `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	StartedAt     *time.Time    `json:"started_at,omitempty"`
	FinishedAt    *time.Time    `json:"finished_at,omitempty"`
}

type datatypesJSON []byte

func (j datatypesJSON) Value() (interface{}, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return []byte(j), nil
}
func (j *datatypesJSON) Scan(v interface{}) error {
	if v == nil {
		*j = []byte("[]")
		return nil
	}
	b, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("invalid json value")
	}
	*j = append((*j)[:0], b...)
	return nil
}

type ImportService struct {
	DB       *gorm.DB
	Notifier notif.Publisher
	mu       sync.Mutex
}

func RegisterImportAPI(router *gin.RouterGroup, db *gorm.DB) {
	s := &ImportService{DB: db, Notifier: notif.NewPublisherFromEnv()}
	router.POST("/users/import-excel", func(c *gin.Context) { s.createJob(c) })
	router.GET("/users/import-excel/:id", func(c *gin.Context) { s.getJob(c) })
}

func (s *ImportService) createJob(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()
	if filepath.Ext(strings.ToLower(header.Filename)) != ".xlsx" {
		c.JSON(400, gin.H{"error": "file must be .xlsx"})
		return
	}
	path := filepath.Join(os.TempDir(), "qrupi-import-"+uuid.NewString()+".xlsx")
	out, err := os.Create(path)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	_, copyErr := io.Copy(out, file)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		c.JSON(500, gin.H{"error": "failed to store import file"})
		return
	}
	id := uuid.New()
	job := UserImportJob{ID: id, UploaderID: c.GetHeader("X-User-ID"), InstitutionID: c.GetHeader("X-Institution-ID"), Status: "pending", Errors: datatypesJSON("[]")}
	if err := s.DB.Create(&job).Error; err != nil {
		_ = os.Remove(path)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	go s.process(path, job)
	c.JSON(http.StatusAccepted, gin.H{"job_id": id, "status": "pending", "message": "Import sedang diproses"})
}

func (s *ImportService) getJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid job_id"})
		return
	}
	var job UserImportJob
	if err := s.DB.First(&job, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "import job not found"})
		return
	}
	c.JSON(200, job)
}

func (s *ImportService) process(path string, job UserImportJob) {
	defer os.Remove(path)
	now := time.Now().UTC()
	s.DB.Model(&job).Updates(map[string]interface{}{"status": "processing", "started_at": now})
	f, err := excelize.OpenFile(path)
	if err != nil {
		s.fail(job, err)
		return
	}
	defer f.Close()
	rows, err := f.GetRows("Users")
	if err != nil || len(rows) < 2 {
		if err == nil {
			err = errors.New("excel file has no data rows")
		}
		s.fail(job, err)
		return
	}
	startRow := 1
	if len(rows) > 1 && strings.Contains(strings.ToLower(strings.Join(rows[1], " ")), "wajib") {
		startRow = 2
	}
	job.TotalData = len(rows) - startRow
	s.DB.Model(&job).Update("total_data", job.TotalData)
	var rowErrors []string
	inserted := 0
	for i := startRow; i < len(rows); i++ {
		if e := s.importRow(rows[i], job.InstitutionID); e != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: %s", i+1, e))
		} else {
			inserted++
		}
		s.DB.Model(&job).Updates(map[string]interface{}{"processed_data": inserted, "failed_data": len(rowErrors)})
	}
	encoded, _ := json.Marshal(rowErrors)
	finished := time.Now().UTC()
	s.DB.Model(&job).Updates(map[string]interface{}{"status": "completed", "errors": datatypesJSON(encoded), "processed_data": inserted, "failed_data": len(rowErrors), "finished_at": finished})
	s.notify(job, "users.import.completed", fmt.Sprintf("Import selesai: %d berhasil, %d gagal", inserted, len(rowErrors)), map[string]interface{}{"job_id": job.ID.String(), "processed_data": inserted, "failed_data": len(rowErrors)})
}

func (s *ImportService) fail(job UserImportJob, err error) {
	finished := time.Now().UTC()
	s.DB.Model(&job).Updates(map[string]interface{}{"status": "failed", "error_message": err.Error(), "finished_at": finished})
	s.notify(job, "users.import.failed", "Import user gagal diproses", map[string]interface{}{"job_id": job.ID.String(), "error": err.Error()})
}

func (s *ImportService) notify(job UserImportJob, typ, msg string, data map[string]interface{}) {
	if job.UploaderID == "" {
		return
	}
	_ = s.Notifier.Publish(context.Background(), notif.Event{Type: notif.EventType(typ), Scope: notif.EventScopeUser, Title: "Import user", Message: msg, UserID: job.UploaderID, InstitutionID: job.InstitutionID, EntityID: job.ID.String(), Data: data, CreatedAt: time.Now().UTC()})
}

func (s *ImportService) importRow(r []string, scopedInstitution string) error {
	get := func(i int) string {
		if i >= len(r) {
			return ""
		}
		return strings.TrimSpace(r[i])
	}
	roleName, name, email, password := get(0), get(1), get(2), get(3)
	institutionCode, phone, pin, contextType, contextCode, isActive, barcodeInput := get(4), get(5), get(6), get(7), get(8), strings.ToLower(get(9)), get(10)
	if name == "" || roleName == "" || isActive == "" {
		return errors.New("name,role_name,status wajib diisi")
	}
	if isActive != "active" && isActive != "inactive" {
		return errors.New("status harus active atau inactive")
	}
	role, err := s.resolveRole(roleName)
	if err != nil {
		return err
	}
	userType := userTypeFromRole(roleName)
	if userType != "student" && email == "" {
		return errors.New("email wajib diisi untuk role selain pelajar")
	}
	if userType == "student" && email == "" {
		email = "student-" + uuid.NewString() + "@internal.qrupi"
	}
	if userType != "student" && password == "" {
		return errors.New("password wajib diisi untuk role selain pelajar")
	}
	var inst *uuid.UUID
	if scopedInstitution != "" {
		id, e := uuid.Parse(scopedInstitution)
		if e != nil {
			return errors.New("invalid institution scope")
		}
		inst = &id
	} else if institutionCode != "" {
		var x models.Institution
		if e := s.DB.Where("LOWER(code)=LOWER(?)", institutionCode).First(&x).Error; e != nil {
			return fmt.Errorf("institution code '%s' tidak ditemukan", institutionCode)
		}
		inst = &x.ID
	}
	var count int64
	s.DB.Model(&models.User{}).Where("email=?", email).Count(&count)
	if count > 0 {
		return fmt.Errorf("email '%s' sudah terpakai", email)
	}
	if userType == "student" && (len(pin) != 6 || strings.Trim(pin, "0123456789") != "") {
		return errors.New("PIN siswa harus tepat 6 digit angka")
	}
	l1 := make([]int, randInt(8)+3)
	for i := range l1 {
		l1[i] = i
	}
	l1s, _ := json.Marshal(l1)
	l2 := randomString(20)
	salt := randomString(12)
	p1 := cryptography.VigenereEncrypt(password, l1)
	p2 := cryptography.PolyalphabetEncrypt(p1, l2)
	pinHash := ""
	barcode := ""
	if userType == "student" {
		h, e := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
		if e != nil {
			return e
		}
		pinHash = string(h)
		barcode = barcodeInput
		if barcode == "" {
			barcode = "QR-" + randomString(12)
		}
		var barcodeCount int64
		s.DB.Model(&models.User{}).Where("barcode = ?", barcode).Count(&barcodeCount)
		if barcodeCount > 0 {
			return fmt.Errorf("barcode '%s' sudah terpakai", barcode)
		}
	}
	return s.DB.Create(&models.User{Name: name, Email: email, RoleID: role.ID, InstitutionID: inst, Type: userType, Phone: phone, ContextType: contextType, ContextCode: contextCode, Status: models.UserStatus(isActive), Password: p2, LayerOne: string(l1s), LayerTwo: l2, Salt: salt, PinHash: pinHash, Barcode: barcode}).Error
}

func (s *ImportService) resolveRole(input string) (models.Role, error) {
	canonical := strings.ToLower(strings.TrimSpace(input))
	switch canonical {
	case "admin", "institution admin", "admin institusi":
		canonical = "institution_admin"
	case "instructor", "instruktur", "guru", "teacher", "pengajar":
		canonical = "instructor"
	case "student", "pelajar", "siswa":
		canonical = "student"
	case "super admin", "superadmin":
		canonical = "super_admin"
	case "dinas pendidikan":
		canonical = "dinas_pendidikan"
	default:
		canonical = strings.ReplaceAll(canonical, " ", "_")
	}

	if roleID := os.Getenv("ROLE_" + strings.ToUpper(canonical) + "_ID"); roleID != "" {
		var role models.Role
		if err := s.DB.Where("id = ?", roleID).First(&role).Error; err == nil {
			return role, nil
		}
	}

	var roles []models.Role
	if err := s.DB.Find(&roles).Error; err != nil {
		return models.Role{}, err
	}
	for _, role := range roles {
		if helpers.IsRole(role.Name, canonical) {
			return role, nil
		}
	}
	return models.Role{}, fmt.Errorf("role '%s' tidak ditemukan", input)
}
func userTypeFromRole(n string) string {
	switch strings.ToLower(strings.TrimSpace(n)) {
	case "student", "siswa":
		return "student"
	case "pelajar":
		return "student"
	case "teacher", "instructor", "instruktur", "pengajar", "guru":
		return "teacher"
	case "admin", "super admin":
		return "admin"
	default:
		return "staff"
	}
}
func randInt(max int) int {
	n, e := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if e != nil {
		return 4
	}
	return int(n.Int64())
}
func randomString(n int) string {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return hex.EncodeToString(make([]byte, n))
	}
	return hex.EncodeToString(b)[:n]
}
