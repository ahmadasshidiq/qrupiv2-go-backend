package learning_resources

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	LearningGroupIDs []string `form:"learning_group_ids" json:"learning_group_ids" binding:"required,min=1"`
	UploadedUserID   string   `form:"uploaded_user_id" json:"uploaded_user_id" binding:"required,uuid"`
	Title            string   `form:"title" json:"title" binding:"required"`
	Description      string   `form:"description" json:"description" binding:"omitempty"`
	Type             string   `form:"type" json:"type" binding:"omitempty,oneof=media interactive-media"`
	Files            []string `form:"files" json:"files" binding:"omitempty"`
}

type UpdateDTO struct {
	LearningGroupIDs []string  `form:"learning_group_ids" json:"learning_group_ids" binding:"omitempty"`
	UploadedUserID   *string   `form:"uploaded_user_id" json:"uploaded_user_id" binding:"omitempty,uuid"`
	Title            *string   `form:"title" json:"title" binding:"omitempty"`
	Description      *string   `form:"description" json:"description" binding:"omitempty"`
	Type             *string   `form:"type" json:"type" binding:"omitempty,oneof=media interactive-media"`
	Files            *[]string `form:"files" json:"files" binding:"omitempty"`
}
