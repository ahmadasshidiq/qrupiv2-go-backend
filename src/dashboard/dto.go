package dashboard

type OverviewDTO struct {
	Months    int    `form:"months" binding:"omitempty,min=1,max=24"`
	StartDate string `form:"start_date" binding:"omitempty,datetime=2006-01-02"`
	EndDate   string `form:"end_date" binding:"omitempty,datetime=2006-01-02"`
}

type Response struct {
	Role      string         `json:"role"`
	RoleLabel string         `json:"role_label"`
	Cached    bool           `json:"cached"`
	Generated string         `json:"generated_at"`
	Summary   map[string]any `json:"summary"`
	Rankings  []any          `json:"rankings"`
	Alerts    []any          `json:"alerts"`
	Data      map[string]any `json:"data"`
}
