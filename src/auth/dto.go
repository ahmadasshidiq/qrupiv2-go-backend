package auth

type RegisterDTO struct {
	Name            string `json:"name" binding:"required,min=2,max=255"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=6"`
	InstitutionName string `json:"institution_name" binding:"required,min=2,max=255"`
	Website         string `json:"website" binding:"omitempty,url,max=255"`
}

type LoginDTO struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

type StudentScanLoginDTO struct {
	QRCode string `json:"qr_code" binding:"required,max=1000"`
}

type StudentVerifyPinDTO struct {
	QRCode string `json:"qr_code" binding:"required,max=1000"`
	Pin    string `json:"pin" binding:"required,numeric,min=4,max=8"`
}

type ResetPasswordDTO struct {
	UserID      string `json:"user_id" binding:"required,uuid"`
	Password    string `json:"password" binding:"required,min=6"`
	ResetByName string `json:"reset_by_name" binding:"omitempty"`
}
