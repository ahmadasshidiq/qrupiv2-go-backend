package attendance_absence_reasons

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	InstitutionID string `json:"institution_id" binding:"omitempty,uuid"`
	Name          string `json:"name" binding:"required,min=2,max=255"`
	Description   string `json:"description" binding:"omitempty"`
}

type UpdateDTO struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description *string `json:"description" binding:"omitempty"`
}
