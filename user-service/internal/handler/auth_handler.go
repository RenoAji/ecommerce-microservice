package handler

import (
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	gojwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required" example:"john@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int    `json:"expires_in" example:"3600"`
	RefreshToken string `json:"refresh_token" example:"1XhvR1bCnUq1UquqTbikKP6vz36-_Ht7dvkSd7P8NTA="`
	TokenType    string `json:"token_type" example:"Bearer"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewAuthMiddleware(repo repository.UserRepository, secretKey string) (*jwt.GinJWTMiddleware, error) {
	// the jwt middleware
	authMiddleware := &jwt.GinJWTMiddleware{
		Realm:      "user-service",
		Key:        []byte("secret key"),
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(c *gin.Context) (any, error) {
			var loginVals struct {
				Email    string `json:"email" binding:"required"`
				Password string `json:"password" binding:"required"`
			}
			if err := c.ShouldBind(&loginVals); err != nil {
				return nil, jwt.ErrMissingLoginValues
			}

			// Find user in DB via repository
			user, err := repo.FindByEmail(loginVals.Email)
			if err != nil {
				return nil, jwt.ErrFailedAuthentication
			}

			// Compare bcrypt passwords
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginVals.Password)); err != nil {
				return nil, jwt.ErrFailedAuthentication
			}

			return user, nil
		},
		PayloadFunc: func(data any) gojwt.MapClaims {
			if v, ok := data.(*domain.User); ok {
				return gojwt.MapClaims{
					"userID":   v.ID,
					"email":    v.Email,
					"username": v.Username,
					"role":     v.Role,
				}
			}
			return gojwt.MapClaims{}
		},
		// this function returns true if the user is allowed to access the resource
		Authorizer: func(c *gin.Context, data any) bool {
			claims := jwt.ExtractClaims(c)

			// Check if required claims exist
			if userID, ok := claims["userID"]; ok && userID != nil {
				return true
			}

			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		TokenLookup: "header:Authorization",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",

		// TokenHeadName is a string in the header. Default value is "Bearer"
		TokenHeadName: "Bearer",

		// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
		TimeFunc: time.Now,
	}

	return authMiddleware, nil
}
