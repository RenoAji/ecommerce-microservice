// internal/domain/claims.go
package domain

import "github.com/golang-jwt/jwt/v5"

type JWTClaims struct {
	UserID   uint   `json:"userID"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"` // Ensure your user-service includes this!
	jwt.RegisteredClaims
}
