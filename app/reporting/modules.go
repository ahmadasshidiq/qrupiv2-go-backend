package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	kafkalib "clasenna-go-backend/libs/kafka"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StudentReadModel struct {
	UserID        string    `gorm:"primaryKey;size:36" json:"user_id"`
	InstitutionID string    `gorm:"index;size:36" json:"institution_id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB) {
	reports := router.Group("/reports")
	reports.GET("/students", func(c *gin.Context) {
		var rows []StudentReadModel
		query := db.WithContext(c.Request.Context())
		if institutionID := c.GetHeader("X-Institution-ID"); institutionID != "" {
			query = query.Where("institution_id = ?", institutionID)
		}
		if err := query.Find(&rows).Error; err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, rows)
	})
}

func HandleEvent(db *gorm.DB, logger *slog.Logger) kafkalib.Handler {
	return func(ctx context.Context, event kafkalib.Envelope) error {
		if event.EventType != "student.created" {
			return nil
		}
		var data struct {
			UserID string `json:"user_id"`
			Name   string `json:"name"`
			Email  string `json:"email"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return err
		}
		model := StudentReadModel{UserID: data.UserID, InstitutionID: event.InstitutionID, Name: data.Name, Email: data.Email, UpdatedAt: time.Now().UTC()}
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&model).Error; err != nil {
			return err
		}
		logger.InfoContext(ctx, "student read model updated", "event_id", event.EventID, "user_id", data.UserID)
		return nil
	}
}
