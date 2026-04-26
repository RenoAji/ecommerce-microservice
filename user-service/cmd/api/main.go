package main

import (
	"libs/logger"
	sharedMiddleware "libs/middleware/gin"
	"net/http"
	"os"
	"time"
	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/domain"
	"user-service/internal/handler"
	"user-service/internal/repository"
	"user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "user-service/docs" // Import generated docs (use your go.mod name)

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title User Service API
// @version 1.0
// @description This is the authentication and user management service for the e-commerce platform.
// @termsOfService http://swagger.io/terms/

// @contact.name Septareno Nugroho Aji
// @contact.email renoaji25sep@gmail.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	gin.DisableConsoleColor()
	// Get database settings from environment variables (provided by docker-compose)
	cfg := config.LoadConfig()
	logger.InitLogger("user-service", cfg.Environment)

	var db *gorm.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{})
		if err == nil {
			break
		}
		logger.Log.Info("Waiting for database", zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		logger.Log.Error("Could not connect to database after retries", zap.Int("attempts", 10), zap.Error(err))
		os.Exit(1)
	}

	// Auto-migrate (creates the table if it doesn't exist)
	db.AutoMigrate(&domain.User{})

	repo := repository.NewPostgresRepository(db)
	svc := service.NewUserService(repo)
	hdl := handler.NewUserHandler(svc)

	// Seed admin user
	database.SeedAdmin(db, svc)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Log.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	authMiddleware, err := handler.NewAuthMiddleware(repo, jwtSecret)
	if err != nil {
		logger.Log.Error("JWT middleware initialization error", zap.Error(err))
		os.Exit(1)
	}

	// Initialize the middleware
	if err := authMiddleware.MiddlewareInit(); err != nil {
		logger.Log.Error("Failed to initialize JWT middleware", zap.Error(err))
		os.Exit(1)
	}

	r := gin.New()
	r.Use(sharedMiddleware.GinLogger())

	// Define Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		api.POST("/register", hdl.Register)
		api.POST("/login", authMiddleware.LoginHandler)

	}

	// Protected Routes (Require JWT)
	auth := r.Group("/api/v1/auth")
	auth.Use(authMiddleware.MiddlewareFunc())
	{
		auth.GET("/profile", hdl.Profile)
		auth.POST("/change-password", hdl.ChangePassword)
		auth.POST("/logout", authMiddleware.LogoutHandler)
	}
	auth.POST("/refresh", authMiddleware.RefreshHandler)

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start Server
	logger.Log.Info("User service starting", zap.String("port", "8081"))
	if err := r.Run(":8081"); err != nil {
		logger.Log.Error("Failed to start server", zap.Error(err))
		os.Exit(1)
	}
}
