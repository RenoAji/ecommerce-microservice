package handler

// This file contains Swagger documentation annotations for Auth routes

import jwt "github.com/appleboy/gin-jwt/v3"

// Login godoc
// @Summary Login to get JWT token
// @Description Authenticate user and return JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 401 {object} ErrorResponse
// @Router /login [post]
func LoginHandler(m *jwt.GinJWTMiddleware) {}

// Logout godoc
// @Summary Log out current user
// @Description Clears the session on the client side. Note: Token remains valid until expiry.
// @Tags Authentication
// @Security Bearer
// @Success 200 {object} map[string]interface{}
// @Router /auth/logout [post]
func LogoutHandler(m *jwt.GinJWTMiddleware) {}

// RefreshToken godoc
// @Summary Refresh JWT token
// @Description Get a new JWT token using the current valid token
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LoginResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func RefreshHandler(m *jwt.GinJWTMiddleware) {}
