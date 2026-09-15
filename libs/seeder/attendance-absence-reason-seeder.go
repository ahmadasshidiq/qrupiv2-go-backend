package seeder

import (
	"errors"

	"clasenna-go-backend/libs/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var defaultAttendanceAbsenceReasons = []struct {
	name        string
	description string
}{
	{name: "Izin", description: "Tidak hadir karena izin"},
	{name: "Alpa", description: "Tidak hadir tanpa keterangan"},
	{name: "Sakit", description: "Tidak hadir karena sakit"},
}

// SeedAttendanceAbsenceReasons creates the default reasons for every institution.
func SeedAttendanceAbsenceReasons(db *gorm.DB) error {
	var institutionIDs []uuid.UUID
	if err := db.Model(&models.Institution{}).Pluck("id", &institutionIDs).Error; err != nil {
		return err
	}

	for _, institutionID := range institutionIDs {
		if err := SeedAttendanceAbsenceReasonsForInstitution(db, institutionID); err != nil {
			return err
		}
	}
	return nil
}

// SeedAttendanceAbsenceReasonsForInstitution is idempotent and restores an
// archived default reason if one already exists.
func SeedAttendanceAbsenceReasonsForInstitution(db *gorm.DB, institutionID uuid.UUID) error {
	for _, item := range defaultAttendanceAbsenceReasons {
		var reason models.AttendanceAbsenceReason
		err := db.Unscoped().Where(
			"institution_id = ? AND name = ?",
			institutionID,
			item.name,
		).First(&reason).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			reason = models.AttendanceAbsenceReason{
				InstitutionID: institutionID,
				Name:          item.name,
				Description:   item.description,
			}
			if err := db.Create(&reason).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if err := db.Unscoped().Model(&reason).Updates(map[string]interface{}{
				"description": item.description,
				"deleted_at":  nil,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
