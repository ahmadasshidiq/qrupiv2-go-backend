package models

import "gorm.io/gorm"

func MigrateLegacyStatusColumns(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable("users") &&
			tx.Migrator().HasColumn("users", "status") &&
			tx.Migrator().HasColumn("users", "is_active") {
			if err := tx.Exec(`UPDATE users
				SET status = is_active
				WHERE is_active IN ('active', 'inactive')`).Error; err != nil {
				return err
			}
		}

		if tx.Migrator().HasTable("learning_groups") &&
			tx.Migrator().HasColumn("learning_groups", "status") &&
			tx.Migrator().HasColumn("learning_groups", "is_active") {
			if err := tx.Exec(`UPDATE learning_groups
				SET status = CASE WHEN is_active THEN 'active' ELSE 'inactive' END`).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
