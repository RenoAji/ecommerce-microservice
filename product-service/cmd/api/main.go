package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"product-service/internal/config"
	"product-service/internal/domain"
	"product-service/internal/handler"
	"product-service/internal/infrastructure"
	"product-service/internal/middleware"
	"product-service/internal/repository"
	"product-service/internal/worker"
	"product-service/pb"
	"syscall"

	"product-service/internal/database"
	"product-service/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"

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
	// Load configuration
	cfg := config.LoadConfig()

	// Set up database connection
	db, err := infrastructure.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Auto-migrate (creates the table if it doesn't exist)
	db.AutoMigrate(&domain.Product{})
	db.AutoMigrate(&domain.Category{})

	// Seed initial data
	database.SeedData(db)

	// Redis Broker Client
	redisBrokerClient := infrastructure.NewRedisBroker(cfg.GetRedisAddr(), cfg.RedisBroker.Password, cfg.RedisBroker.DB)

	// Repository Service, and handlers
	repo := repository.NewPostgresRepository(db)
	eventRepo := repository.NewRedisRepository(redisBrokerClient)
	svc := service.NewProductService(repo, eventRepo)
	ProductHandler := handler.NewProductHandler(svc)
	CategoryHandler := handler.NewCategoryHandler(svc)
	
	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis Consumer Group
	err = infrastructure.InitProductConsumerGroup(ctx, redisBrokerClient)
	if err != nil {
		log.Fatalf("Failed to initialize consumer group: %v", err)
	}
    

	// Worker for consuming order messages
	orderWorker := worker.NewOrderWorker(redisBrokerClient, svc)
	go orderWorker.ListenForOrders(ctx)

	stockInsufficientWorker := worker.NewPaymentFailedWorker(redisBrokerClient, svc)
	go stockInsufficientWorker.ListenForPaymentFailures(ctx)

	r := gin.Default()

	// Define Routes
	api := r.Group("/api/v1")
	{
		// admin only routes
		adminRoutes := api.Group("/")
		adminRoutes.Use(middleware.AdminMiddleware())
		{
			adminRoutes.POST("/products", ProductHandler.Create)
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

	// Start gRPC Server in a goroutine
	var grpcServer *grpc.Server
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Failed to listen on port 50051: %v", err)
		}
		
		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewProductGRPCServer(svc)
		pb.RegisterProductServiceServer(grpcServer, grpcHandler)
		
		log.Println("gRPC server starting on port 50051...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	// Start HTTP Server in a goroutine
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}
	
	go func() {
		log.Println("Product Service starting on port " + cfg.ServerPort + "...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	<-quit
	log.Println("Shutting down servers...")

	// Cancel context to stop worker
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	// Shutdown gRPC server
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	log.Println("Servers gracefully stopped")
}
