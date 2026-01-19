package main

import (
	"fmt"
	"log"
	"os"
	"product-service/internal/domain"
	"product-service/internal/handler"
	"product-service/internal/middleware"
	"product-service/internal/repository"

	"product-service/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "product-service/docs"
)

// @title Product Service API
// @version 1.0
// @description This is the product catalog and inventory management service for the e-commerce platform.
// @termsOfService http://swagger.io/terms/

// @contact.name Septareno Nugroho Aji
// @contact.email renoaji25sep@gmail.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8082
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
	db.AutoMigrate(&domain.Product{})
	db.AutoMigrate(&domain.Category{})

	repo := repository.NewPostgresRepository(db)
	svc := service.NewProductService(repo)
	ProductHandler := handler.NewProductHandler(svc)
	CategoryHandler := handler.NewCategoryHandler(svc)

	r := gin.Default()

	// Define Routes
	api := r.Group("/api/v1")
	{
		// admin only routes
		adminRoutes := api.Group("/")
		adminRoutes.Use(middleware.AdminMiddleware())
		{
			adminRoutes.POST("/products", ProductHandler.Create)
			adminRoutes.PATCH("/products/:id/stock", ProductHandler.AddStock)
			adminRoutes.PUT("/products/:id", ProductHandler.Update)
			adminRoutes.DELETE("/products/:id", ProductHandler.Delete)
			adminRoutes.POST("/categories", CategoryHandler.Create)
		}

		// public routes
		api.GET("/products", ProductHandler.Get)
		api.GET("/products/:id", ProductHandler.GetByID)
	}

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start Server
	log.Println("Product Service starting on port 8081...")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
