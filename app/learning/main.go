package main

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	loglib "clasenna-go-backend/libs/logger"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/stores"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"os/signal"
	"syscall"
)

func main() {
	_ = godotenv.Load()
	logger := loglib.New("learning-service")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := stores.OpenPostgres("LEARNING")
	if err != nil {
		logger.Error("database startup failed", "error", err)
		return
	}
	if err := db.AutoMigrate(
		&models.LearningGroup{},
		&models.LearningGroupMember{},
		&models.LearningResource{},
		&models.LearningResourceGroup{},
		&models.Quiz{},
		&models.QuizSession{},
		&models.AttendanceAbsenceReason{},
		&models.AttendanceLog{},
	); err != nil {
		logger.Error("database migration failed", "error", err)
		return
	}
	if err := migrateLearningResourceGroups(db); err != nil {
		logger.Error("learning resource migration failed", "error", err)
		return
	}
	if err := migrateLegacyAttendanceLogs(db); err != nil {
		logger.Error("attendance migration failed", "error", err)
		return
	}
	r := gin.New()
	r.Use(gin.Recovery())
	metrics := &httpserver.Metrics{}
	r.Use(httpserver.Middleware(logger, metrics))
	httpserver.RegisterProbes(r, metrics, func(ctx context.Context) error {
		sqlDB, e := db.DB()
		if e != nil {
			return e
		}
		return sqlDB.PingContext(ctx)
	})
	RegisterAllModules(r.Group("/api/v1"), db)
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("LEARNING_PORT", "3003"), r, logger); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
