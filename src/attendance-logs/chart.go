package attendance_logs

import (
	"errors"
	"sort"
	"time"

	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
)

type chartRecord struct {
	Date, Status, UserID, UserName, GroupID, GroupName string
	LateMinutes                                        float64
}

type AttendanceChart struct {
	Summary            ChartSummary      `json:"summary"`
	DailyTrend         []ChartDailyTrend `json:"daily_trend"`
	AttendanceByStatus []ChartStatus     `json:"attendance_by_status"`
	ByLearningGroup    []ChartGroup      `json:"by_learning_group"`
	TopAttendance      []ChartUser       `json:"top_attendance"`
	LowestAttendance   []ChartUser       `json:"lowest_attendance"`
}
type ChartSummary struct {
	TotalRecords       int     `json:"total_records"`
	OnTime             int     `json:"on_time"`
	Late               int     `json:"late"`
	Absent             int     `json:"absent"`
	AttendanceRate     float64 `json:"attendance_rate"`
	AverageLateMinutes float64 `json:"average_late_minutes"`
}
type ChartDailyTrend struct {
	Date   string `json:"date"`
	OnTime int    `json:"on_time"`
	Late   int    `json:"late"`
	Absent int    `json:"absent"`
	Total  int    `json:"total"`
}
type ChartStatus struct {
	Status     string  `json:"status"`
	Label      string  `json:"label"`
	Total      int     `json:"total"`
	Percentage float64 `json:"percentage"`
}
type ChartGroup struct {
	LearningGroupID   string  `json:"learning_group_id"`
	LearningGroupName string  `json:"learning_group_name"`
	Total             int     `json:"total"`
	OnTime            int     `json:"on_time"`
	Late              int     `json:"late"`
	Absent            int     `json:"absent"`
	AttendanceRate    float64 `json:"attendance_rate"`
}
type ChartUser struct {
	UserID         string  `json:"user_id"`
	UserName       string  `json:"user_name"`
	Total          int     `json:"total"`
	OnTime         int     `json:"on_time"`
	Late           int     `json:"late"`
	Absent         int     `json:"absent"`
	AttendanceRate float64 `json:"attendance_rate"`
}

func (s *AttendanceLogService) chart(ctx *gin.Context, dto ChartDTO) (*AttendanceChart, error) {
	start, _ := time.Parse("2006-01-02", dto.StartDate)
	end, _ := time.Parse("2006-01-02", dto.EndDate)
	if end.Before(start) {
		return nil, errors.New("end_date must be on or after start_date")
	}
	if dto.Type == "teacher" && dto.LearningGroupID != "" {
		return nil, errors.New("learning_group_id is only valid for student type")
	}

	q := s.DB.WithContext(ctx.Request.Context()).Table("attendance_logs al").
		Select("to_char(al.occurred_at::date, 'YYYY-MM-DD') as date, al.status, al.user_id, u.name as user_name, COALESCE(al.learning_group_id::text, '') as group_id, COALESCE(lg.name, '') as group_name, CASE WHEN al.status = 'late' AND al.check_in_at IS NOT NULL THEN GREATEST(EXTRACT(EPOCH FROM (al.check_in_at - al.occurred_at)) / 60, 0) ELSE 0 END as late_minutes").
		Joins("JOIN users u ON u.id = al.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN learning_groups lg ON lg.id = al.learning_group_id AND lg.deleted_at IS NULL").
		Where("al.type = ? AND al.occurred_at >= ? AND al.occurred_at < ? AND al.deleted_at IS NULL", dto.Type, start, end.AddDate(0, 0, 1))
	if institutionID := ctx.GetString("institution_id"); institutionID != "" {
		q = q.Where("u.institution_id = ?", institutionID)
	}
	if dto.LearningGroupID != "" {
		q = q.Where("al.learning_group_id = ?", dto.LearningGroupID)
	}
	if dto.UserID != "" {
		q = q.Where("al.user_id = ?", dto.UserID)
	}
	var rows []chartRecord
	if err := q.Order("al.occurred_at, al.user_id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := &AttendanceChart{DailyTrend: []ChartDailyTrend{}, AttendanceByStatus: []ChartStatus{}, ByLearningGroup: []ChartGroup{}, TopAttendance: []ChartUser{}, LowestAttendance: []ChartUser{}}
	daily := map[string]*ChartDailyTrend{}
	groups := map[string]*ChartGroup{}
	users := map[string]*ChartUser{}
	for _, r := range rows {
		result.Summary.TotalRecords++
		switch r.Status {
		case string(models.AttendanceStatusOnTime):
			result.Summary.OnTime++
		case string(models.AttendanceStatusLate):
			result.Summary.Late++
			result.Summary.AverageLateMinutes += r.LateMinutes
		case string(models.AttendanceStatusAbsent):
			result.Summary.Absent++
		}
		d := daily[r.Date]
		if d == nil {
			d = &ChartDailyTrend{Date: r.Date}
			daily[r.Date] = d
		}
		d.Total++
		incrementStatus(&d.OnTime, &d.Late, &d.Absent, r.Status)
		if r.GroupID != "" {
			g := groups[r.GroupID]
			if g == nil {
				g = &ChartGroup{LearningGroupID: r.GroupID, LearningGroupName: r.GroupName}
				groups[r.GroupID] = g
			}
			g.Total++
			incrementStatus(&g.OnTime, &g.Late, &g.Absent, r.Status)
		}
		u := users[r.UserID]
		if u == nil {
			u = &ChartUser{UserID: r.UserID, UserName: r.UserName}
			users[r.UserID] = u
		}
		u.Total++
		incrementStatus(&u.OnTime, &u.Late, &u.Absent, r.Status)
	}
	if result.Summary.TotalRecords > 0 {
		result.Summary.AttendanceRate = percentage(result.Summary.OnTime+result.Summary.Late, result.Summary.TotalRecords)
	}
	if result.Summary.Late > 0 {
		result.Summary.AverageLateMinutes = round(result.Summary.AverageLateMinutes / float64(result.Summary.Late))
	}
	for _, d := range daily {
		result.DailyTrend = append(result.DailyTrend, *d)
	}
	sort.Slice(result.DailyTrend, func(i, j int) bool { return result.DailyTrend[i].Date < result.DailyTrend[j].Date })
	for _, g := range groups {
		g.AttendanceRate = percentage(g.OnTime+g.Late, g.Total)
		result.ByLearningGroup = append(result.ByLearningGroup, *g)
	}
	sort.Slice(result.ByLearningGroup, func(i, j int) bool {
		return result.ByLearningGroup[i].LearningGroupName < result.ByLearningGroup[j].LearningGroupName
	})
	for _, u := range users {
		u.AttendanceRate = percentage(u.OnTime+u.Late, u.Total)
		result.TopAttendance = append(result.TopAttendance, *u)
		result.LowestAttendance = append(result.LowestAttendance, *u)
	}
	sort.Slice(result.TopAttendance, func(i, j int) bool {
		return result.TopAttendance[i].AttendanceRate > result.TopAttendance[j].AttendanceRate
	})
	if len(result.TopAttendance) > 5 {
		result.TopAttendance = result.TopAttendance[:5]
	}
	sort.Slice(result.LowestAttendance, func(i, j int) bool {
		return result.LowestAttendance[i].AttendanceRate < result.LowestAttendance[j].AttendanceRate
	})
	if len(result.LowestAttendance) > 5 {
		result.LowestAttendance = result.LowestAttendance[:5]
	}
	labels := map[string]string{"on_time": "Tepat waktu", "late": "Terlambat", "absent": "Tidak hadir"}
	for _, st := range []string{"on_time", "late", "absent"} {
		n := 0
		if st == "on_time" {
			n = result.Summary.OnTime
		}
		if st == "late" {
			n = result.Summary.Late
		}
		if st == "absent" {
			n = result.Summary.Absent
		}
		result.AttendanceByStatus = append(result.AttendanceByStatus, ChartStatus{Status: st, Label: labels[st], Total: n, Percentage: percentage(n, result.Summary.TotalRecords)})
	}
	return result, nil
}
func incrementStatus(on, late, absent *int, status string) {
	switch status {
	case "on_time":
		*on++
	case "late":
		*late++
	case "absent":
		*absent++
	}
}
func percentage(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return round(float64(n) * 100 / float64(total))
}
func round(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
