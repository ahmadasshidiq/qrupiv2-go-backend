package cryptography

import (
	"clasenna-go-backend/libs/models"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func GenerateToken(userID string, email string, roleID string, expiration time.Duration) (string, error) {
	return GenerateTokenWithInstitution(userID, email, roleID, "", expiration)
}

func GenerateTokenWithInstitution(userID string, email string, roleID string, institutionID string, expiration time.Duration) (string, error) {
	now := time.Now().UTC()

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role_id": roleID,
		"iat":     now.Unix(),
	}
	if institutionID != "" {
		claims["institution_id"] = institutionID
	}

	if expiration > 0 {
		claims["exp"] = time.Now().Add(expiration).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))
}

func ValidateToken(tokenString string) (*jwt.Token, jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})

	if err != nil {
		return nil, nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, nil, fmt.Errorf("invalid token")
	}
	return token, claims, nil
}

func JWTMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Set("auth_error", "Authorization header is required")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.Replace(authHeader, "Bearer", "", 1))
		token, claims, err := ValidateToken(tokenString)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if user.CurrentToken != tokenString {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is no longer active"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role_id", user.RoleID.String())
		if user.InstitutionID != nil {
			c.Set("institution_id", user.InstitutionID.String())
		}
		c.Next()
	}
}
