package roles

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type PermissionItem struct {
	Model  string `json:"model" binding:"required"`
	Action string `json:"action" binding:"required"`
}

type CreateDTO struct {
	Name        string           `json:"name" binding:"required,max=100"`
	Permissions []PermissionItem `json:"permissions" binding:"omitempty,dive"`
	Description string           `json:"description" binding:"omitempty"`
}

type UpdateDTO struct {
	Name        *string           `json:"name" binding:"omitempty,max=100"`
	Permissions *[]PermissionItem `json:"permissions" binding:"omitempty,dive"`
	Description *string           `json:"description" binding:"omitempty"`
}
