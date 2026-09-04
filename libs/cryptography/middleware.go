package cryptography

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccessMiddleware struct {
	DB *gorm.DB
}

func (m *AccessMiddleware) Handler(modelName string, action string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		roleIDVal, exists := ctx.Get("role_id")
		if !exists {
			res := helpers.FormatResponse(ctx.Request.Method, "auth", http.StatusBadRequest, "Role ID is missing", nil, nil)
			ctx.JSON(http.StatusUnauthorized, res)
			ctx.Abort()
			return
		}

		roleID, err := uuid.Parse(roleIDVal.(string))
		if err != nil {
			res := helpers.FormatResponse(ctx.Request.Method, "auth", http.StatusBadRequest, nil, nil, err)
			ctx.JSON(http.StatusBadRequest, res)
			ctx.Abort()
			return
		}

		var role models.Role
		if err := m.DB.First(&role, "id = ?", roleID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				res := helpers.FormatResponse(ctx.Request.Method, "auth", http.StatusForbidden, "Role not found", nil, nil)
				ctx.JSON(http.StatusForbidden, res)
				ctx.Abort()
				return
			}
			res := helpers.FormatResponse(ctx.Request.Method, "auth", http.StatusInternalServerError, "Database error", nil, err)
			ctx.JSON(http.StatusInternalServerError, res)
			ctx.Abort()
			return
		}

		// Decode JSON permissions
		var permissions []models.PermissionItem
		if err := json.Unmarshal(role.Permissions, &permissions); err != nil {
			res := helpers.FormatResponse(ctx.Request.Method, "auth", http.StatusInternalServerError, "Invalid permission data", nil, err)
			ctx.JSON(http.StatusInternalServerError, res)
			ctx.Abort()
			return
		}

		// Cek apakah permission cocok
		allowed := false
		for _, p := range permissions {
			if p.Model == modelName && p.Action == action {
				allowed = true
				break
			}
		}

		if !allowed {
			res := helpers.FormatResponse(ctx.Request.Method, "auth", http.StatusForbidden, "Access denied", nil, nil)
			ctx.JSON(http.StatusForbidden, res)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
