package middleware

import (
	"net/http"
	"os"
	"strings"

	"delivery-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)
func AdminMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &domain.JWTClaims{}

		//  Parse and Validate the token
		jwtSecret := os.Getenv("JWT_SECRET")
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		
		// Check the Role
		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Admins only"})
			c.Abort()
			return
		}

		// 3. Store user info in context
		c.Set("userID", claims.UserID)

		c.Next()
    }
}