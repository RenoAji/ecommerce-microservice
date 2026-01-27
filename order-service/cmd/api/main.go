package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"order-service/internal/domain"
	"order-service/internal/handler"
	"order-service/internal/middleware"
	"order-service/internal/repository"
	"order-service/internal/service"
	"order-service/pb"

	_ "order-service/docs"
)

// @title Order Service API
// @version 1.0
// @description This is the order management service for the e-commerce platform.
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
	db.AutoMigrate(&domain.Order{}, &domain.OrderItem{})


	// Set up gRPC connection to Cart Service
	conn, err := grpc.NewClient("cart-service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
	CartClient := pb.NewCartServiceClient(conn)

	// Set up gRPC connection to Product Service
	conn, err = grpc.NewClient("product-service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
	ProductClient := pb.NewProductServiceClient(conn)

	repo := repository.NewPostgresRepository(db)
	svc := service.NewOrderService(repo, CartClient, ProductClient)
	hdl := handler.NewOrderHandler(svc)
	
	// register routes
	r := gin.Default()

	api:= r.Group("/api/v1")
	{
		order:= api.Group("/order")
		order.Use(middleware.AuthMiddleware())
		{
			order.POST("", hdl.PostOrder)
			order.GET("", hdl.GetOrders)
			order.GET("/:id", hdl.GetOrderByID)

		}
	}

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start Server
	log.Println("Order Service starting on port 8081...")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}