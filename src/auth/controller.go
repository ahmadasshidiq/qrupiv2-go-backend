package auth

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	service *AuthService
}

func NewAuthController(service *AuthService) *AuthController {
	return &AuthController{service}
}

// @Summary      Register user
// @Tags         Auth API
// @Param        dto  body  RegisterDTO  true  "Register Data"
// @Router       /auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var dto RegisterDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "register", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	data, err := c.service.Register(ctx.Request.Context(), dto, ctx.GetString("correlation_id"))
	if err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "register", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := helpers.FormatResponse(ctx.Request.Method, "register", http.StatusCreated, data, nil, nil)
	ctx.JSON(http.StatusCreated, res)
}

// @Summary      Login user
// @Tags         Auth API
// @Param        dto  body  LoginDTO  true  "Login Data"
// @Router       /auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var dto LoginDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "login", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	data, err := c.service.Login(ctx.Request.Context(), dto, ctx.GetString("correlation_id"))
	if err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "login", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := helpers.FormatResponse(ctx.Request.Method, "login", http.StatusOK, data, nil, nil)
	ctx.JSON(http.StatusOK, res)
}

// @Summary      Login student using barcode and PIN
// @Tags         Auth API
// @Param        dto  body  StudentScanLoginDTO  true  "Student Scan Login Data"
// @Router       /auth/student/scan [post]
func (c *AuthController) StudentScanLogin(ctx *gin.Context) {
	var dto StudentScanLoginDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "student-scan-login", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	data, err := c.service.StudentScanLogin(ctx.Request.Context(), dto, ctx.GetString("correlation_id"))
	if err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "student-scan-login", http.StatusUnauthorized, nil, nil, err)
		ctx.JSON(http.StatusUnauthorized, res)
		return
	}

	res := helpers.FormatResponse(ctx.Request.Method, "student-scan-login", http.StatusOK, data, nil, nil)
	ctx.JSON(http.StatusOK, res)
}

// @Summary      Verify student PIN after QR scan
// @Tags         Auth API
// @Param        dto  body  StudentVerifyPinDTO  true  "Student PIN Data"
// @Router       /auth/student/verify-pin [post]
func (c *AuthController) StudentVerifyPin(ctx *gin.Context) {
	var dto StudentVerifyPinDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, helpers.FormatResponse(ctx.Request.Method, "student-verify-pin", http.StatusBadRequest, nil, nil, err))
		return
	}
	data, err := c.service.StudentVerifyPin(ctx.Request.Context(), dto, ctx.GetString("correlation_id"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, helpers.FormatResponse(ctx.Request.Method, "student-verify-pin", http.StatusUnauthorized, nil, nil, err))
		return
	}
	ctx.JSON(http.StatusOK, helpers.FormatResponse(ctx.Request.Method, "student-verify-pin", http.StatusOK, data, nil, nil))
}

// @Summary      Reset password user
// @Tags         Auth API
// @Param        dto  body  ResetPasswordDTO  true  "Reset Password Data"
// @Router       /auth/reset-password [post]
func (c *AuthController) ResetPassword(ctx *gin.Context) {
	var dto ResetPasswordDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "reset-password", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	if err := c.service.ResetPassword(ctx.Request.Context(), dto, ctx.GetString("correlation_id")); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "reset-password", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	res := helpers.FormatResponse(ctx.Request.Method, "reset-password", http.StatusOK, gin.H{
		"message": "password updated",
	}, nil, nil)
	ctx.JSON(http.StatusOK, res)
}
