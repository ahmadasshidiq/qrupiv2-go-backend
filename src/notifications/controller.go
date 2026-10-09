package notifications

import (
	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/models"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type Controller struct{ service *Service }

func NewController(s *Service) *Controller { return &Controller{s} }
func (c *Controller) Register(r *gin.RouterGroup) {
	g := r.Group("/notifications")
	g.Use(cryptography.JWTMiddleware(c.service.DB))
	protected := r.Group("")
	protected.Use(cryptography.JWTMiddleware(c.service.DB))
	g.GET("", c.List)
	g.GET("/unread-count", c.UnreadCount)
	g.GET("/:id", c.Get)
	g.GET("/announcements/:id", c.GetAnnouncement)
	g.PATCH("/:id/read", c.MarkRead)
	g.PATCH("/read-all", c.MarkAllRead)
	protected.POST("/notifications/devices", c.RegisterDevice)
	protected.GET("/notifications/announcements", c.ListAnnouncements)
	protected.POST("/notifications/announcements", c.CreateAnnouncement)
}
func (c *Controller) List(ctx *gin.Context) {
	var rows []models.Notification
	q := c.service.DB.Where("user_id = ?", ctx.GetString("user_id"))
	if ctx.Query("unread") == "true" {
		q = q.Where("read_at IS NULL")
	}
	limit, page := 20, 1
	if v, e := strconv.Atoi(ctx.Query("limit")); e == nil && v > 0 && v <= 100 {
		limit = v
	}
	if v, e := strconv.Atoi(ctx.Query("page")); e == nil && v > 0 {
		page = v
	}
	var total int64
	q.Model(&models.Notification{}).Count(&total)
	if e := q.Order("created_at desc").Limit(limit).Offset((page - 1) * limit).Find(&rows).Error; e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(200, gin.H{"data": rows, "page": page, "limit": limit, "total": total})
}
func (c *Controller) UnreadCount(ctx *gin.Context) {
	var n int64
	c.service.DB.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", ctx.GetString("user_id")).Count(&n)
	ctx.JSON(200, gin.H{"count": n})
}
func (c *Controller) Get(ctx *gin.Context) {
	id, e := uuid.Parse(ctx.Param("id"))
	if e != nil {
		ctx.JSON(400, gin.H{"error": "invalid notification id"})
		return
	}
	var n models.Notification
	e = c.service.DB.Where("id=? AND user_id=? AND deleted_at IS NULL", id, ctx.GetString("user_id")).First(&n).Error
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			ctx.JSON(404, gin.H{"error": "notification not found"})
		} else {
			ctx.JSON(500, gin.H{"error": e.Error()})
		}
		return
	}
	ctx.JSON(200, n)
}

func (c *Controller) GetAnnouncement(ctx *gin.Context) {
	id, e := uuid.Parse(ctx.Param("id"))
	if e != nil {
		ctx.JSON(400, gin.H{"error": "invalid announcement id"})
		return
	}

	var a models.Announcement
	q := c.service.DB.Where("id = ? AND deleted_at IS NULL", id)
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		q = q.Where("institution_id = ? OR audience = ?", institutionID, models.AnnouncementAudienceSystem)
	}
	if e = q.First(&a).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			ctx.JSON(404, gin.H{"error": "announcement not found"})
		} else {
			ctx.JSON(500, gin.H{"error": e.Error()})
		}
		return
	}
	ctx.JSON(200, a)
}
func (c *Controller) MarkRead(ctx *gin.Context) {
	id, e := uuid.Parse(ctx.Param("id"))
	if e != nil {
		ctx.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	e = c.service.DB.Model(&models.Notification{}).Where("id=? AND user_id=?", id, ctx.GetString("user_id")).Update("read_at", time.Now()).Error
	if e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
		return
	}
	ctx.Status(204)
}
func (c *Controller) MarkAllRead(ctx *gin.Context) {
	e := c.service.DB.Model(&models.Notification{}).Where("user_id=? AND read_at IS NULL", ctx.GetString("user_id")).Update("read_at", time.Now()).Error
	if e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
		return
	}
	ctx.Status(204)
}
func (c *Controller) RegisterDevice(ctx *gin.Context) {
	var d DeviceDTO
	if ctx.ShouldBindJSON(&d) != nil {
		ctx.JSON(400, gin.H{"error": "token is required"})
		return
	}
	uid, e := uuid.Parse(ctx.GetString("user_id"))
	if e != nil {
		ctx.JSON(401, gin.H{"error": "invalid user"})
		return
	}
	row := models.NotificationDevice{UserID: uid, Token: d.Token, Platform: d.Platform, IsActive: true, LastSeenAt: time.Now()}
	if e = c.service.DB.Where("token = ?", d.Token).Assign(row).FirstOrCreate(&row).Error; e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(201, row.ID)
}
func (c *Controller) ListAnnouncements(ctx *gin.Context) {
	var rows []models.Announcement
	q := c.service.DB.Where("deleted_at IS NULL")
	if id := ctx.GetString("institution_id"); id != "" {
		q = q.Where("institution_id=? OR audience=?", id, models.AnnouncementAudienceSystem)
	}
	if e := q.Order("created_at DESC").Limit(100).Find(&rows).Error; e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(200, gin.H{"data": rows})
}
func (c *Controller) CreateAnnouncement(ctx *gin.Context) {
	var d AnnouncementDTO
	if ctx.ShouldBindJSON(&d) != nil || d.Title == "" || d.Message == "" {
		ctx.JSON(400, gin.H{"error": "title and message are required"})
		return
	}
	creator, e := uuid.Parse(ctx.GetString("user_id"))
	if e != nil {
		ctx.JSON(401, gin.H{"error": "invalid user"})
		return
	}
	a := models.Announcement{CreatedBy: creator, Audience: d.Audience, Title: d.Title, Message: d.Message}
	if v := ctx.GetString("institution_id"); v != "" {
		id, _ := uuid.Parse(v)
		a.InstitutionID = &id
	}
	if d.TargetID != nil {
		id, _ := uuid.Parse(*d.TargetID)
		a.TargetID = &id
	}
	if e = c.service.DB.Create(&a).Error; e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
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
			c.service.DB.Table("learning_group_members").Where("learning_group_id=? AND role_in_group='student' AND deleted_at IS NULL", *a.TargetID).Pluck("user_id", &ids)
		}
	case models.AnnouncementAudienceInstitution:
		c.service.DB.Model(&models.User{}).Where("institution_id=? AND deleted_at IS NULL", a.InstitutionID).Pluck("id", &ids)
	default:
		c.service.DB.Model(&models.User{}).Where("deleted_at IS NULL").Pluck("id", &ids)
	}
	rec := make([]string, 0, len(ids))
	for _, id := range ids {
		rec = append(rec, id.String())
	}
	if e = c.service.Persist(ctx, string("announcement.created"), ctx.GetString("institution_id"), rec, d.Title, d.Message, "", "", "", "", nil); e != nil {
		ctx.JSON(500, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(201, a.ID)
}
