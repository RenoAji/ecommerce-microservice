package handler

import (
	"net/http"
	"strings"
	"user-service/internal/domain"
	"user-service/internal/service"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(us *service.UserService) *UserHandler {
	return &UserHandler{userService: us}
}

func (h *UserHandler) Register(c *gin.Context) {
	var user domain.User

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Call the service layer
	if err := h.userService.RegisterUser(&user); err != nil {
		// Check PostgreSQL unique constraint violation return 409
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, gin.H{"error": "email or username already exists"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func (h *UserHandler) Profile(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	c.JSON(200, gin.H{"userID": claims["userID"], "email": claims["email"], "username": claims["username"]})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	claims := jwt.ExtractClaims(c)
	userID := uint(claims["userID"].(float64))

	// Call the service layer, pass the userID from JWT claims and passwords from request
	if err := h.userService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		if strings.Contains(err.Error(), "incorrect old password") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect old password"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not change password"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
