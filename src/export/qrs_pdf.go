package export

import (
	"clasenna-go-backend/libs/barcodes"
	"clasenna-go-backend/libs/models"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

func (s *ExportService) generateQRS(institutionID, learningGroupID string) (string, error) {
	var institution models.Institution
	if err := s.DB.Select("name").First(&institution, "id = ?", institutionID).Error; err != nil {
		return "", err
	}
	learningGroupName := "Semua Learning Group"
	if learningGroupID != "" {
		var group models.LearningGroup
		if err := s.DB.Select("name").First(&group, "id = ? AND institution_id = ?", learningGroupID, institutionID).Error; err != nil {
			return "", err
		}
		learningGroupName = group.Name
	}
	var users []models.User
	q := s.DB.Where("users.institution_id = ? AND users.deleted_at IS NULL AND users.barcode <> ''", institutionID).Order("users.name ASC")
	if learningGroupID != "" {
		if _, err := uuid.Parse(learningGroupID); err != nil {
			return "", errors.New("invalid learning_group_id format")
		}
		q = q.Joins("JOIN learning_group_members lgm ON lgm.user_id = users.id AND lgm.learning_group_id = ? AND lgm.deleted_at IS NULL", learningGroupID)
	}
	if err := q.Find(&users).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	if len(users) == 0 {
		return "", errors.New("no users with QR data found")
	}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("QR Siswa", false)
	for i, user := range users {
		if i%12 == 0 {
			pdf.AddPage()
			pdf.SetFont("Arial", "B", 13)
			pdf.CellFormat(0, 7, institution.Name, "", 1, "C", false, 0, "")
			pdf.SetFont("Arial", "", 10)
			pdf.CellFormat(0, 6, "Learning Group: "+learningGroupName, "", 1, "C", false, 0, "")
		}
		position := i % 12
		col, row := position%3, position/3
		x, y := 10+float64(col)*65, 35+float64(row)*65
		pdf.SetLineWidth(.4)
		pdf.Rect(x, y, 60, 60, "D")
		image, err := barcodes.QRPNG(user.Barcode, 180)
		if err != nil {
			return "", err
		}
		imagePath := filepath.Join(os.TempDir(), fmt.Sprintf("qr_%s.png", user.ID))
		if err := os.WriteFile(imagePath, image, 0600); err != nil {
			return "", err
		}
		pdf.ImageOptions(imagePath, x+6, y+4, 48, 48, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
		_ = os.Remove(imagePath)
		pdf.SetFont("Arial", "B", 10)
		pdf.SetFont("Arial", "", 6)
		pdf.SetXY(x+3, y+53)
		pdf.CellFormat(54, 4, user.Name, "", 0, "C", false, 0, "")
	}
	path := filepath.Join(os.TempDir(), "student-qrs_"+time.Now().Format("20060102150405")+".pdf")
	if err := pdf.OutputFileAndClose(path); err != nil {
		return "", err
	}
	return path, nil
}
