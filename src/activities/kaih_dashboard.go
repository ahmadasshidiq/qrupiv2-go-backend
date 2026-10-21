package activities

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type KAIHDashboardDTO struct {
	StartDate       string `form:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate         string `form:"end_date" binding:"required,datetime=2006-01-02"`
	LearningGroupID string `form:"learning_group_id" binding:"omitempty,uuid"`
}

type KAIHStudentRow struct {
	No         int              `json:"no"`
	UserID     uuid.UUID        `json:"user_id"`
	Name       string           `json:"nama"`
	Categories map[string]int64 `json:"kategori"`
	Average    float64          `json:"rata_rata_nilai"`
	Predicate  string           `json:"predikat"`
}

type KAIHDashboardResponse struct {
	StartDate          string           `json:"start_date"`
	EndDate            string           `json:"end_date"`
	Days               int              `json:"jumlah_hari"`
	MinimumPerCategory int              `json:"minimum_per_kategori"`
	Students           []KAIHStudentRow `json:"students"`
}

type kaihAggregate struct {
	UserID             uuid.UUID
	UserName, ItemName string
	Points             int64
}

func (s *ActivityService) getKAIHDashboard(ctx *gin.Context, dto KAIHDashboardDTO) (*KAIHDashboardResponse, error) {
	start, err := time.Parse("2006-01-02", dto.StartDate)
	if err != nil {
		return nil, errors.New("invalid start_date format")
	}
	end, err := time.Parse("2006-01-02", dto.EndDate)
	if err != nil {
		return nil, errors.New("invalid end_date format")
	}
	if end.Before(start) {
		return nil, errors.New("start_date cannot be later than end_date")
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 7 {
		return nil, errors.New("date range must be at least 7 days")
	}
	minPoints := int(math.Ceil(float64(days) * 0.7))

	var items []models.ActivityItem
	q := s.DB.Where("deleted_at IS NULL AND type = ?", models.ActivityItemTypePositive).
		Where("category_id IN (SELECT id FROM activity_categories WHERE name = ? AND deleted_at IS NULL)", "7 KAIH").Order("name ASC")
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		q = q.Where("institution_id = ?", institutionID)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != 7 {
		return nil, errors.New("7 KAIH must contain exactly 7 activity items")
	}
	itemNames := make([]string, len(items))
	itemIDs := make([]uuid.UUID, len(items))
	for i, item := range items {
		itemNames[i], itemIDs[i] = item.Name, item.ID
	}

	usersQ := s.DB.Table("users u").Where("u.deleted_at IS NULL AND u.type = ?", "student")
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		usersQ = usersQ.Where("u.institution_id = ?", institutionID)
	}
	if dto.LearningGroupID != "" {
		usersQ = usersQ.Joins("JOIN learning_group_members lgm ON lgm.user_id = u.id AND lgm.deleted_at IS NULL AND lgm.learning_group_id = ?", dto.LearningGroupID)
	}
	type userRow struct {
		ID   uuid.UUID
		Name string
	}
	var users []userRow
	if err := usersQ.Select("u.id, u.name").Order("u.name ASC").Scan(&users).Error; err != nil {
		return nil, err
	}
	var aggregates []kaihAggregate
	actQ := s.DB.Table("activities a").Joins("JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL").Joins("JOIN activity_items ai ON ai.id = a.item_id AND ai.deleted_at IS NULL").Where("a.deleted_at IS NULL AND a.item_id IN ? AND a.occurred_at >= ? AND a.occurred_at < ?", itemIDs, start, end.AddDate(0, 0, 1))
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		actQ = actQ.Where("u.institution_id = ?", institutionID)
	}
	if dto.LearningGroupID != "" {
		actQ = actQ.Where("EXISTS (SELECT 1 FROM learning_group_members x WHERE x.user_id = a.user_id AND x.learning_group_id = ? AND x.deleted_at IS NULL)", dto.LearningGroupID)
	}
	if err := actQ.Select("a.user_id, u.name AS user_name, ai.name AS item_name, COALESCE(SUM(a.point_value),0) AS points").Group("a.user_id, u.name, ai.name").Scan(&aggregates).Error; err != nil {
		return nil, err
	}
	points := map[uuid.UUID]map[string]int64{}
	for _, a := range aggregates {
		if points[a.UserID] == nil {
			points[a.UserID] = map[string]int64{}
		}
		points[a.UserID][a.ItemName] = a.Points
	}
	result := &KAIHDashboardResponse{StartDate: dto.StartDate, EndDate: dto.EndDate, Days: days, MinimumPerCategory: minPoints, Students: make([]KAIHStudentRow, 0, len(users))}
	for i, u := range users {
		cats := map[string]int64{}
		var total int64
		all := true
		for _, name := range itemNames {
			v := points[u.ID][name]
			cats[name] = v
			total += v
			if v < int64(minPoints) {
				all = false
			}
		}
		result.Students = append(result.Students, KAIHStudentRow{No: i + 1, UserID: u.ID, Name: u.Name, Categories: cats, Average: float64(total) / 7, Predicate: map[bool]string{true: "Terbiasa", false: "Belum Terbiasa"}[all]})
	}
	sort.Slice(result.Students, func(i, j int) bool { return result.Students[i].Name < result.Students[j].Name })
	for i := range result.Students {
		result.Students[i].No = i + 1
	}
	return result, nil
}

func (c *ActivityController) GetKAIHDashboard(ctx *gin.Context) {
	var dto KAIHDashboardDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "activities", http.StatusBadRequest, err)
		return
	}
	r, err := c.Service.getKAIHDashboard(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "activities", http.StatusBadRequest, err)
		return
	}
	helpers.RespondSuccess(ctx, "activities", http.StatusOK, r)
}
