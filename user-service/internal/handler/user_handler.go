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

// RegisterRequest represents the registration payload
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Username string `json:"username" binding:"required,min=3" example:"johndoe"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

// SuccessResponse represents a success message
type SuccessResponse struct {
	Message string `json:"message"`
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account in the system
// @Tags Authentication
// @Accept json
// @Produce json
// @Param user body RegisterRequest true "User Registration Data"
// @Success 201 {object} SuccessResponse "User registered successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body: validation error details"
// @Failure 409 {object} ErrorResponse "Email or username already exists"
// @Failure 500 {object} ErrorResponse "Internal server error while creating user"
// @Router /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var user domain.User

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		return
	}

	// Call the service layer
	if err := h.userService.RegisterUser(&user); err != nil {
		// Check PostgreSQL unique constraint violation return 409
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, ErrorResponse{Code: http.StatusConflict, Message: "Email or username already exists"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: http.StatusInternalServerError, Message: "Internal server error while creating user"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, SuccessResponse{Message: "User registered successfully"})
}

// Profile godoc
// @Summary Get user profile
// @Description Get current user profile information
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "userID, email, username"
// @Failure 401 {object} ErrorResponse
// @Router /auth/profile [get]
func (h *UserHandler) Profile(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	c.JSON(200, gin.H{"userID": claims["userID"], "email": claims["email"], "username": claims["username"]})
}

// ChangePassword godoc
// @Summary Change user password
// @Description Change the password for the authenticated user
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param passwords body object{old_password=string,new_password=string} true "Old and new passwords"
// @Success 200 {object} SuccessResponse "Password changed successfully"
// @Failure 400 {object} ErrorResponse "Invalid request: passwords are required and must be at least 6 characters"
// @Failure 401 {object} ErrorResponse "Incorrect old password"
// @Failure 500 {object} ErrorResponse "Internal server error while changing password"
// @Router /auth/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required,min=6"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: http.StatusBadRequest, Message: "Invalid request: passwords are required and must be at least 6 characters"})
		return
	}

	claims := jwt.ExtractClaims(c)
	userID := uint(claims["userID"].(float64))

	// Call the service layer, pass the userID from JWT claims and passwords from request
	if err := h.userService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		if strings.Contains(err.Error(), "incorrect old password") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Code: http.StatusUnauthorized, Message: "Incorrect old password"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: http.StatusInternalServerError, Message: "Internal server error while changing password"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, SuccessResponse{Message: "Password changed successfully"})
}
