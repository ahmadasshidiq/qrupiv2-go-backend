package institutions

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	CurrentSubscriptionID string  `form:"current_subscription_id" binding:"omitempty,uuid"`
	Name                  string  `form:"name" binding:"required,max=255"`
	AvatarURL             string  `form:"avatar_url" binding:"omitempty,url,max=255"`
	Address               string  `form:"address" binding:"omitempty,max=255"`
	Latitude              float64 `form:"latitude" binding:"omitempty,latitude"`
	Longitude             float64 `form:"longitude" binding:"omitempty,longitude"`
	Phone                 string  `form:"phone" binding:"omitempty,max=20"`
	Website               string  `form:"website" binding:"omitempty,url,max=255"`
	ProvinceCode          string  `form:"province_code" binding:"omitempty,max=20"`
	RegencyCode           string  `form:"regency_code" binding:"omitempty,max=20"`
	DistrictCode          string  `form:"district_code" binding:"omitempty,max=20"`
	VillageCode           string  `form:"village_code" binding:"omitempty,max=20"`
	Country               string  `form:"country" binding:"omitempty,max=100"`
	ZipCode               string  `form:"zip_code" binding:"omitempty,max=20"`
	Status                string  `form:"status" binding:"omitempty,oneof=active inactive"`
	Email                 string  `form:"email" binding:"required,email"`
}

type UpdateDTO struct {
	CurrentSubscriptionID *string  `form:"current_subscription_id" binding:"omitempty,uuid"`
	Name                  *string  `form:"name" binding:"omitempty,max=255"`
	AvatarURL             *string  `form:"avatar_url" binding:"omitempty,url,max=255"`
	Address               *string  `form:"address" binding:"omitempty,max=255"`
	Latitude              *float64 `form:"latitude" binding:"omitempty,latitude"`
	Longitude             *float64 `form:"longitude" binding:"omitempty,longitude"`
	Phone                 *string  `form:"phone" binding:"omitempty,max=20"`
	Website               *string  `form:"website" binding:"omitempty,url,max=255"`
	ProvinceCode          *string  `form:"province_code" binding:"omitempty,max=20"`
	RegencyCode           *string  `form:"regency_code" binding:"omitempty,max=20"`
	DistrictCode          *string  `form:"district_code" binding:"omitempty,max=20"`
	VillageCode           *string  `form:"village_code" binding:"omitempty,max=20"`
	Country               *string  `form:"country" binding:"omitempty,max=100"`
	ZipCode               *string  `form:"zip_code" binding:"omitempty,max=20"`
	Status                *string  `form:"status" binding:"omitempty,oneof=active inactive"`
}
