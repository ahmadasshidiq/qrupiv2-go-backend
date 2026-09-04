package main

import "gorm.io/gorm"

func migrateInstitutionRegionSnapshot(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		legacyNames := []struct {
			column string
			target string
		}{
			{"province", "province_name"},
			{"city", "regency_name"},
			{"district", "district_name"},
			{"sub_district", "village_name"},
		}
		for _, field := range legacyNames {
			if !tx.Migrator().HasColumn("institutions", field.column) {
				continue
			}
			query := "UPDATE institutions SET " + field.target + " = " + field.column +
				" WHERE (" + field.target + " IS NULL OR " + field.target + " = '')" +
				" AND " + field.column + " IS NOT NULL AND " + field.column + " <> ''"
			if err := tx.Exec(query).Error; err != nil {
				return err
			}
		}

		if !tx.Migrator().HasTable("provinces") || !tx.Migrator().HasTable("regencies") ||
			!tx.Migrator().HasTable("districts") || !tx.Migrator().HasTable("villages") {
			return nil
		}
		queries := []string{
			`UPDATE institutions i SET province_code = p.code
			 FROM provinces p WHERE i.province_code = '' AND i.province_name = p.name AND p.is_active = true`,
			`UPDATE institutions i SET regency_code = r.code
			 FROM regencies r WHERE i.regency_code = '' AND i.regency_name = r.name
			 AND r.province_code = i.province_code AND r.is_active = true`,
			`UPDATE institutions i SET district_code = d.code
			 FROM districts d WHERE i.district_code = '' AND i.district_name = d.name
			 AND d.regency_code = i.regency_code AND d.is_active = true`,
			`UPDATE institutions i SET village_code = v.code
			 FROM villages v WHERE i.village_code = '' AND i.village_name = v.name
			 AND v.district_code = i.district_code AND v.is_active = true`,
		}
		for _, query := range queries {
			if err := tx.Exec(query).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
