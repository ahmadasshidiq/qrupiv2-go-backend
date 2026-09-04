package dashboard

type DashboardOverviewDTO struct {
	Months    int    `form:"months"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}
