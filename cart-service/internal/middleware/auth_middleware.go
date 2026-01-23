package middleware

import (
	"net/http"
	"os"
	"strings"

	"cart-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 1. Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &domain.JWTClaims{}

		// 2. Parse and Validate the token
		jwtSecret := os.Getenv("JWT_SECRET")
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 3. Store user info in context
		c.Set("userID", claims.UserID)

		c.Next()
    }
}