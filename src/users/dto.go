package users

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	Name          string `form:"name" binding:"required,min=2,max=255"`
	Email         string `form:"email" binding:"omitempty,email"`
	Password      string `form:"password" binding:"omitempty,min=6"`
	Type          string `form:"type" binding:"omitempty,oneof=student teacher admin staff"`
	Pin           string `form:"pin" binding:"omitempty,numeric,min=4,max=8"`
	RoleID        string `form:"role_id" binding:"required,uuid"`
	InstitutionID string `form:"institution_id" binding:"omitempty,uuid"`
	Phone         string `form:"phone" binding:"omitempty,max=20"`
	ContextCode   string `form:"context_code" binding:"omitempty,max=100"`
	ContextType   string `form:"context_type" binding:"omitempty,max=50"`
	Status        string `form:"status" binding:"required,oneof=active inactive"`
	AvatarURL     string `form:"avatar_url" binding:"omitempty,url"`
}

type UpdateDTO struct {
	Name          *string `form:"name" binding:"omitempty,min=2,max=255"`
	Email         *string `form:"email" binding:"omitempty,email"`
	Password      *string `form:"password" binding:"omitempty,min=6"`
	Type          *string `form:"type" binding:"omitempty,oneof=student teacher admin staff"`
	Pin           *string `form:"pin" binding:"omitempty,numeric,min=4,max=8"`
	RoleID        *string `form:"role_id" binding:"omitempty,uuid"`
	InstitutionID *string `form:"institution_id" binding:"omitempty,uuid"`
	Phone         *string `form:"phone" binding:"omitempty,max=20"`
	ContextType   *string `form:"context_type" binding:"omitempty,max=50"`
	ContextCode   *string `form:"context_code" binding:"omitempty,max=100"`
	Status        *string `form:"status" binding:"omitempty,oneof=active inactive"`
	AvatarURL     *string `form:"avatar_url" binding:"omitempty,url"`
}
