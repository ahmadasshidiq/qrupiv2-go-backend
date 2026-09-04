package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	loglib "clasenna-go-backend/libs/logger"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/stores"
	"clasenna-go-backend/src/regions"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger := loglib.New("region-service")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := stores.OpenPostgres("REGION")
	if err != nil {
		logger.Error("database startup failed", "error", err)
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("database pool setup failed", "error", err)
		return
	}
	sqlDB.SetMaxOpenConns(helpers.ConfigInt("REGION_DB_MAX_OPEN_CONNS", 5))
	sqlDB.SetMaxIdleConns(helpers.ConfigInt("REGION_DB_MAX_IDLE_CONNS", 2))
	if err := db.AutoMigrate(
		&models.Province{},
		&models.Regency{},
		&models.District{},
		&models.Village{},
		&models.RegionSyncState{},
	); err != nil {
		logger.Error("database migration failed", "error", err)
		return
	}

	synchronizer := regions.NewSynchronizer(
		db,
		logger,
		helpers.ConfigString("REGION_API_BASE_URL", "https://wilayah.id/api"),
		helpers.ConfigInt("REGION_SYNC_CONCURRENCY", 8),
	)
	synchronizer.Client.Timeout = helpers.ConfigDuration("REGION_SYNC_REQUEST_TIMEOUT", 30*time.Second)
	go regions.RunScheduler(
		ctx,
		synchronizer,
		helpers.ConfigInt("REGION_SYNC_INTERVAL_MONTHS", 3),
		helpers.ConfigDuration("REGION_SYNC_RETRY_DELAY", 15*time.Minute),
	)

	router := gin.New()
	router.Use(gin.Recovery())
	metrics := &httpserver.Metrics{}
	router.Use(httpserver.Middleware(logger, metrics))
	httpserver.RegisterProbes(router, metrics, func(ctx context.Context) error {
		return sqlDB.PingContext(ctx)
	})
	RegisterAllModules(router.Group("/api/v1"), db)
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("REGION_PORT", "3007"), router, logger); err != nil {
		logger.Error("service stopped", "error", err)
	}
}
