package dashboard

type DashboardController struct {
	Service *DashboardService
}

func NewController(service *DashboardService) *DashboardController {
	return &DashboardController{Service: service}
}
