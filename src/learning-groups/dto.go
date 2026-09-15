package learning_groups

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	Name          string `json:"name" binding:"required,min=2,max=100"`
	InstitutionID string `json:"institution_id" binding:"required,uuid"`
	Code          string `json:"code" binding:"required"`
	Type          string `json:"type" binding:"required,oneof=school-class university-class club"`
	Level         int    `json:"level" binding:"required"`
	Major         string `json:"major" binding:"omitempty,max=100"`
	Department    string `json:"department" binding:"omitempty,max=100"`
	AcademicYear  string `json:"academic_year" binding:"omitempty,len=9"`
	Status        string `json:"status" binding:"required,oneof=active inactive"`
}

type UpdateDTO struct {
	Name          *string `json:"name" binding:"omitempty,min=2,max=100"`
	InstitutionID *string `json:"institution_id" binding:"omitempty,uuid"`
	Code          *string `json:"code" binding:"omitempty"`
	Type          *string `json:"type" binding:"omitempty,oneof=school-class university-class club"`
	Level         *int    `json:"level" binding:"omitempty"`
	Major         *string `json:"major" binding:"omitempty,max=100"`
	Department    *string `json:"department" binding:"omitempty,max=100"`
	AcademicYear  *string `json:"academic_year" binding:"omitempty,len=9"`
	Status        *string `json:"status" binding:"omitempty,oneof=active inactive"`
}
