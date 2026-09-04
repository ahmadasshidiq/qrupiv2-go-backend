package activityitems

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
}
type CreateDTO struct {
	CategoryID  string `json:"category_id" binding:"omitempty,uuid"`
	Name        string `json:"name" binding:"required,min=2,max=255"`
	Description string `json:"description" binding:"omitempty"`
	Type        string `json:"type" binding:"required,oneof=positive violation"`
	PointValue  int    `json:"point_value" binding:"omitempty,min=0"`
	DailyLimit  int    `json:"daily_limit" binding:"omitempty,min=0"`
	PeriodLimit int    `json:"period_limit" binding:"omitempty,min=0"`
	PeriodType  string `json:"period_type" binding:"omitempty,oneof=none weekly monthly lifetime"`
	IsSendNotif bool   `json:"is_send_notif"`
	Color       string `json:"color" binding:"omitempty,max=20"`
}
type UpdateDTO struct {
	CategoryID  *string `json:"category_id" binding:"omitempty,uuid"`
	Name        *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description *string `json:"description" binding:"omitempty"`
	Type        *string `json:"type" binding:"omitempty,oneof=positive violation"`
	PointValue  *int    `json:"point_value" binding:"omitempty,min=0"`
	DailyLimit  *int    `json:"daily_limit" binding:"omitempty,min=0"`
	PeriodLimit *int    `json:"period_limit" binding:"omitempty,min=0"`
	PeriodType  *string `json:"period_type" binding:"omitempty,oneof=none weekly monthly lifetime"`
	IsSendNotif *bool   `json:"is_send_notif"`
	Color       *string `json:"color" binding:"omitempty,max=20"`
}
