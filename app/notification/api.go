package main

import (
	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

func RegisterNotificationAPI(r *gin.RouterGroup, db *gorm.DB) {
	g := r.Group("/notifications")
	g.Use(cryptography.JWTMiddleware(db))
	protected := r.Group("")
	protected.Use(cryptography.JWTMiddleware(db))
	g.GET("", func(c *gin.Context) {
		var rows []models.Notification
		q := db.Where("user_id = ?", c.GetString("user_id"))
		if v := c.Query("unread"); v == "true" {
			q = q.Where("read_at IS NULL")
		}
		limit := 20
		page := 1
		if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= 100 {
			limit = v
		}
		if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
			page = v
		}
		var total int64
		q.Model(&models.Notification{}).Count(&total)
		if err := q.Order("created_at desc").Limit(limit).Offset((page - 1) * limit).Find(&rows).Error; err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"data": rows, "page": page, "limit": limit, "total": total})
	})
	g.GET("/unread-count", func(c *gin.Context) {
		var n int64
		db.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", c.GetString("user_id")).Count(&n)
		c.JSON(200, gin.H{"count": n})
	})
	g.PATCH("/:id/read", func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		now := time.Now()
		res := db.Model(&models.Notification{}).Where("id=? AND user_id=?", id, c.GetString("user_id")).Updates(map[string]interface{}{"read_at": now})
		if res.Error != nil {
			c.JSON(500, gin.H{"error": res.Error.Error()})
			return
		}
		c.Status(204)
	})
	g.PATCH("/read-all", func(c *gin.Context) {
		now := time.Now()
		if e := db.Model(&models.Notification{}).Where("user_id=? AND read_at IS NULL", c.GetString("user_id")).Update("read_at", now).Error; e != nil {
			c.JSON(500, gin.H{"error": e.Error()})
			return
		}
		c.Status(204)
	})
	protected.POST("/notifications/devices", func(c *gin.Context) {
		var d struct{ Token, Platform string }
		if c.ShouldBindJSON(&d) != nil || d.Token == "" {
			c.JSON(400, gin.H{"error": "token is required"})
			return
		}
		uid, _ := uuid.Parse(c.GetString("user_id"))
		row := models.NotificationDevice{UserID: uid, Token: d.Token, Platform: d.Platform, IsActive: true, LastSeenAt: time.Now()}
		if e := db.Where("token = ?", d.Token).Assign(row).FirstOrCreate(&row).Error; e != nil {
			c.JSON(500, gin.H{"error": e.Error()})
			return
		}
		c.JSON(201, row.ID)
	})
	protected.GET("/notifications/announcements", func(c *gin.Context) {
		var rows []models.Announcement
		q := db.Where("deleted_at IS NULL")
		if institutionID := c.GetString("institution_id"); institutionID != "" {
			q = q.Where("institution_id = ? OR audience = ?", institutionID, models.AnnouncementAudienceSystem)
		}
		if err := q.Order("created_at DESC").Limit(100).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": rows})
	})
	protected.POST("/notifications/announcements", func(c *gin.Context) {
		var d struct {
			Audience       models.AnnouncementAudience `json:"audience"`
			TargetID       *string                     `json:"target_id"`
			Title, Message string
		}
		if c.ShouldBindJSON(&d) != nil || d.Title == "" || d.Message == "" {
			c.JSON(400, gin.H{"error": "title and message are required"})
			return
		}
		creator, e := uuid.Parse(c.GetString("user_id"))
		if e != nil {
			c.JSON(401, gin.H{"error": "invalid user"})
			return
		}
		a := models.Announcement{CreatedBy: creator, Audience: d.Audience, Title: d.Title, Message: d.Message}
		if v := c.GetString("institution_id"); v != "" {
			id, _ := uuid.Parse(v)
			a.InstitutionID = &id
		}
		if d.TargetID != nil {
			id, _ := uuid.Parse(*d.TargetID)
			a.TargetID = &id
		}
		if e := db.Create(&a).Error; e != nil {
			c.JSON(500, gin.H{"error": e.Error()})
			return
		}
		var ids []uuid.UUID
		switch d.Audience {
		case models.AnnouncementAudienceUser:
			if a.TargetID != nil {
				ids = []uuid.UUID{*a.TargetID}
			}
		case models.AnnouncementAudienceLearningGroup:
			if a.TargetID != nil {
				db.Table("learning_group_members").Where("learning_group_id=? AND role_in_group='student' AND deleted_at IS NULL", *a.TargetID).Pluck("user_id", &ids)
			}
		case models.AnnouncementAudienceInstitution:
			db.Model(&models.User{}).Where("institution_id=? AND deleted_at IS NULL", a.InstitutionID).Pluck("id", &ids)
		default:
			db.Model(&models.User{}).Where("deleted_at IS NULL").Pluck("id", &ids)
		}
		rec := make([]string, 0, len(ids))
		for _, id := range ids {
			rec = append(rec, id.String())
		}
		if e := persistNotifications(c, db, nil, nil, "announcement.created", c.GetString("institution_id"), rec, d.Title, d.Message, "", "", "", "", nil); e != nil {
			c.JSON(500, gin.H{"error": e.Error()})
			return
		}
		c.JSON(201, a.ID)
	})
}

var _ = http.StatusOK
