package main

import (
	"clasenna-go-backend/libs/models"
	"gorm.io/gorm"
)

func migrateLearningResourceGroups(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasColumn(&models.LearningResource{}, "learning_group_id") {
		return nil
	}
	if err := db.Exec(`INSERT INTO learning_resource_groups (learning_resource_id, learning_group_id)
		SELECT id, learning_group_id FROM learning_resources
		WHERE learning_group_id IS NOT NULL
		ON CONFLICT (learning_resource_id, learning_group_id) DO NOTHING`).Error; err != nil {
		return err
	}
	return migrator.DropColumn(&models.LearningResource{}, "learning_group_id")
}

func migrateLegacyAttendanceLogs(db *gorm.DB) error {
	migrator := db.Migrator()
	if err := migrateLegacyAbsenceReasons(db); err != nil {
		return err
	}
	if migrator.HasColumn(&models.AttendanceLog{}, "role_in_group") {
		if err := db.Exec(`UPDATE attendance_logs
			SET type = CASE WHEN role_in_group = 'instructor' THEN 'teacher' ELSE 'student' END
			WHERE type IS NULL OR type = '' OR type = 'student'`).Error; err != nil {
			return err
		}
	}
	if migrator.HasColumn(&models.AttendanceLog{}, "location_lat") {
		if err := db.Exec(`UPDATE attendance_logs
			SET check_in_at = COALESCE(check_in_at, created_at),
				check_in_lat = COALESCE(check_in_lat, location_lat),
				check_in_long = COALESCE(check_in_long, location_long)
			WHERE status IN ('on_time', 'late')`).Error; err != nil {
			return err
		}
	}
	for _, column := range []string{"role_in_group", "location_lat", "location_long"} {
		if migrator.HasColumn(&models.AttendanceLog{}, column) {
			if err := migrator.DropColumn(&models.AttendanceLog{}, column); err != nil {
				return err
			}
		}
	}
	return nil
}

func migrateLegacyAbsenceReasons(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.AttendanceLog{}) {
		return nil
	}
	if err := db.Exec(`INSERT INTO attendance_absence_reasons
			(id, institution_id, name, description, created_at, updated_at)
		SELECT gen_random_uuid(), source.institution_id, source.name, source.description, NOW(), NOW()
		FROM (
			SELECT DISTINCT u.institution_id,
				CASE al.status WHEN 'permission' THEN 'Izin' WHEN 'sick' THEN 'Sakit' ELSE 'Alpa' END AS name,
				CASE al.status WHEN 'permission' THEN 'Tidak hadir karena izin' WHEN 'sick' THEN 'Tidak hadir karena sakit' ELSE 'Tidak hadir tanpa keterangan' END AS description
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			WHERE al.status IN ('absent', 'permission', 'sick') AND u.institution_id IS NOT NULL
		) source
		ON CONFLICT (institution_id, name) DO NOTHING`).Error; err != nil {
		return err
	}
	return db.Exec(`UPDATE attendance_logs al
		SET absence_reason_id = reason.id, status = 'absent', requires_check_out = false
		FROM users u
		JOIN attendance_absence_reasons reason ON reason.institution_id = u.institution_id
		WHERE u.id = al.user_id
			AND al.status IN ('absent', 'permission', 'sick')
			AND reason.name = CASE al.status WHEN 'permission' THEN 'Izin' WHEN 'sick' THEN 'Sakit' ELSE 'Alpa' END
			AND al.absence_reason_id IS NULL`).Error
}
