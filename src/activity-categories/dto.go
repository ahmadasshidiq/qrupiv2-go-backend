package activitycategories

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
}
type CreateDTO struct {
	Name        string `json:"name" binding:"required,min=2,max=255"`
	Description string `json:"description" binding:"omitempty"`
	Color       string `json:"color" binding:"omitempty,max=20"`
}
type UpdateDTO struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description *string `json:"description" binding:"omitempty"`
	Color       *string `json:"color" binding:"omitempty,max=20"`
}
