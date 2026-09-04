package regions

type FindDTO struct {
	Search       string `form:"search" binding:"omitempty,max=255"`
	ProvinceCode string `form:"province_code" binding:"omitempty,max=20"`
	RegencyCode  string `form:"regency_code" binding:"omitempty,max=20"`
	DistrictCode string `form:"district_code" binding:"omitempty,max=20"`
	SortBy       string `form:"sortBy" binding:"omitempty,oneof=code name created_at updated_at"`
	SortOrder    string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit        int    `form:"limit" binding:"omitempty,min=1,max=500"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
}
