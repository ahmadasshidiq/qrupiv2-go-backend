package stores

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitPostgres() *gorm.DB {
	db, err := OpenPostgres("CORE")
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	DB = db
	log.Println("Successfully connected to PostgreSQL")
	return db
}

func OpenPostgres(service string) (*gorm.DB, error) {
	nodeEnv := os.Getenv("NODE_ENV")
	prefix := strings.ToUpper(service)
	dsnProd := os.Getenv(prefix + "_POSTGRES_DSN_PROD")
	dsnDev := os.Getenv(prefix + "_POSTGRES_DSN_DEV")
	if dsnProd == "" {
		dsnProd = os.Getenv("POSTGRES_DSN_PROD")
	}
	if dsnDev == "" {
		dsnDev = os.Getenv("POSTGRES_DSN_DEV")
	}
	dsn := dsnDev
	if nodeEnv == "production" {
		dsn = dsnProd
	}

	newLogger := logger.Default.LogMode(logger.Silent)
	if nodeEnv != "production" {
		newLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to %s PostgreSQL: %w", service, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get %s SQL database: %w", service, err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
