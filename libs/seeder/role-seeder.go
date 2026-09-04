package seeder

import (
	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/models"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"time"

	"gorm.io/gorm"
)

func SeedRoles(DB *gorm.DB) error {
	permissionsJSON, err := json.Marshal(MasterPermissions)
	if err != nil {
		return err
	}

	role := models.Role{
		Name:        "Super Admin",
		Permissions: permissionsJSON,
	}

	var existing models.Role
	err = DB.Where("name = ?", role.Name).First(&existing).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		if err := DB.Create(&role).Error; err != nil {
			return err
		}
		log.Println("[Seeder] ✅ Super Admin role created")

	case err != nil:
		return err

	default:
		if string(existing.Permissions) != string(permissionsJSON) {
			existing.Permissions = permissionsJSON
			if err := DB.Save(&existing).Error; err != nil {
				return err
			}
			log.Println("[Seeder] 🔄 Super Admin permissions updated")
		} else {
			log.Println("[Seeder] ℹ️ Super Admin role already up-to-date")
		}
	}

	return nil
}

// Seeder untuk user Super Admin (owner)
func SeedSuperAdminUser(DB *gorm.DB) error {
	adminEmail := os.Getenv("SEED_EMAIL")
	adminPassword := os.Getenv("SEED_PASS")

	// pastikan role Super Admin sudah ada
	var superAdminRole models.Role
	if err := DB.Where("name = ?", "Super Admin").First(&superAdminRole).Error; err != nil {
		return errors.New("Super Admin role not found — jalankan SeedRoles terlebih dahulu")
	}

	// cek apakah user sudah ada
	var existing models.User
	if err := DB.Where("email = ?", adminEmail).First(&existing).Error; err == nil {
		log.Println("[Seeder] ℹ️ Super Admin user already exists")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// generate salt dan layer
	rand.Seed(time.Now().UnixNano())
	layerOne := rand.Perm(rand.Intn(6) + 3)
	layerTwo := cryptography.GenerateRandomString(rand.Intn(25) + 12)
	salt := cryptography.GenerateRandomString(rand.Intn(12) + 8)

	layerOneStr, _ := json.Marshal(layerOne)

	// enkripsi password
	encodedOne := cryptography.VigenereEncrypt(adminPassword, layerOne)
	encodedTwo := cryptography.PolyalphabetEncrypt(encodedOne, layerTwo)

	user := models.User{
		Name:     "Owner",
		Email:    adminEmail,
		RoleID:   superAdminRole.ID,
		Password: encodedTwo,
		Salt:     salt,
		LayerOne: string(layerOneStr),
		LayerTwo: layerTwo,
		IsActive: models.UserStatusActive,
	}

	if err := DB.Create(&user).Error; err != nil {
		return err
	}

	log.Printf("[Seeder] ✅ Super Admin user created (email: %s)", adminEmail)
	return nil
}

func RunSeeders(DB *gorm.DB) {
	seeders := []func(*gorm.DB) error{
		SeedRoles,
		SeedSuperAdminUser,
	}

	for _, seed := range seeders {
		if err := seed(DB); err != nil {
			log.Fatalf("[Seeder] ❌ Failed to run seeder: %v", err)
		}
	}

	log.Println("[Seeder] 🎉 All seeders completed successfully!")
}
