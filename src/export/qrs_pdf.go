package export

import (
	"bytes"
	"clasenna-go-backend/libs/barcodes"
	"clasenna-go-backend/libs/models"
	"errors"
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

	const (
		perPage  = 12
		boxSize  = 60.0
		gap      = 5.0
		startX   = 10.0
		startY   = 30.0
		qrSize   = 48.0
		nameBoxW = 54.0
	)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("QR Siswa", false)
	pdf.SetAutoPageBreak(false, 0) // paginasi diatur manual, jangan biarkan gofpdf memecah halaman
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	for i, user := range users {
		if i%perPage == 0 {
			pdf.AddPage()
			pdf.SetFont("Arial", "B", 13)
			pdf.SetXY(10, 10)
			pdf.CellFormat(0, 7, tr(institution.Name), "", 1, "C", false, 0, "")
			pdf.SetFont("Arial", "", 10)
			pdf.CellFormat(0, 6, tr("Learning Group: "+learningGroupName), "", 1, "C", false, 0, "")
		}

		position := i % perPage
		col, row := position%3, position/3
		x := startX + float64(col)*(boxSize+gap)
		y := startY + float64(row)*(boxSize+gap)

		pdf.SetLineWidth(.4)
		pdf.Rect(x, y, boxSize, boxSize, "D")

		image, err := barcodes.QRPNG(user.Barcode, 512)
		if err != nil {
			return "", err
		}
		imgName := "qr_" + user.ID.String() // sesuaikan kalau user.ID bertipe string: "qr_" + user.ID
		pdf.RegisterImageOptionsReader(imgName, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(image))
		pdf.ImageOptions(imgName, x+(boxSize-qrSize)/2, y+4, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

		// Nama: perkecil font jika terlalu panjang
		name := tr(user.Name)
		fontSize := 7.0
		pdf.SetFont("Arial", "", fontSize)
		for fontSize > 4 && pdf.GetStringWidth(name) > nameBoxW {
			fontSize -= 0.5
			pdf.SetFont("Arial", "", fontSize)
		}
		pdf.SetXY(x+(boxSize-nameBoxW)/2, y+boxSize-6)
		pdf.CellFormat(nameBoxW, 4, name, "", 0, "C", false, 0, "")
	}

	path := filepath.Join(os.TempDir(), "student-qrs_"+time.Now().Format("20060102150405")+".pdf")
	if err := pdf.OutputFileAndClose(path); err != nil {
		return "", err
	}
	return path, nil
}
