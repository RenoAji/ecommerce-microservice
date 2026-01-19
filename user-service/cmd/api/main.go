package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"user-service/internal/database"
	"user-service/internal/domain"
	"user-service/internal/handler"
	"user-service/internal/repository"
	"user-service/internal/service"

	"github.com/gin-gonic/gin"
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

// @host localhost:8081
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Get database settings from environment variables (provided by docker-compose)
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, dbname, port)
	var db *gorm.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Waiting for database... attempt %d", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Could not connect to database after 10 attempts")
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
		log.Fatal("JWT_SECRET environment variable is required")
	}

	authMiddleware, err := handler.NewAuthMiddleware(repo, jwtSecret)
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	// Initialize the middleware
	if err := authMiddleware.MiddlewareInit(); err != nil {
		log.Fatal("Failed to initialize JWT middleware: ", err)
	}

	r := gin.Default()

	// Define Routes
	api := r.Group("/api/v1")
	{
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
	log.Println("User Service starting on port 8081...")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
