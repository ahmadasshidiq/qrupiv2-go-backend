package legal_documents

type DefaultFindDTO struct {
	Search    string `form:"search"`
	Slug      string `form:"slug" binding:"omitempty,max=100"`
	Status    string `form:"status" binding:"omitempty,oneof=draft published archived"`
	SortBy    string `form:"sortBy" binding:"omitempty,oneof=slug title version status effective_date created_at updated_at"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	Slug          string `json:"slug" binding:"required,max=100"`
	Title         string `json:"title" binding:"required,max=255"`
	Content       string `json:"content" binding:"required"`
	Version       string `json:"version" binding:"required,max=50"`
	Status        string `json:"status" binding:"required,oneof=draft published archived"`
	EffectiveDate string `json:"effective_date" binding:"required"`
}

type UpdateDTO struct {
	Slug          *string `json:"slug" binding:"omitempty,max=100"`
	Title         *string `json:"title" binding:"omitempty,max=255"`
	Content       *string `json:"content"`
	Version       *string `json:"version" binding:"omitempty,max=50"`
	Status        *string `json:"status" binding:"omitempty,oneof=draft published archived"`
	EffectiveDate *string `json:"effective_date"`
}
