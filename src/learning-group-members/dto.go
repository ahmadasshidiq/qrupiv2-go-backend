package learning_group_members

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	LearningGroupID string `json:"learning_group_id" binding:"required,uuid"`
	UserID          string `json:"user_id" binding:"required,uuid"`
	RoleInGroup     string `json:"role_in_group" binding:"required,oneof=instructor student"`
}

type UpdateDTO struct {
	LearningGroupID *string `json:"learning_group_id" binding:"omitempty,uuid"`
	UserID          *string `json:"user_id" binding:"omitempty,uuid"`
	RoleInGroup     *string `json:"role_in_group" binding:"omitempty,oneof=instructor student"`
}
