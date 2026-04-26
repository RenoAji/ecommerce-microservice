package main

import (
	"context"
	"fmt"
	"libs/logger"
	sharedMiddleware "libs/middleware/gin"
	"libs/pb"
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
	"syscall"

	"product-service/internal/database"
	"product-service/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"libs/consulclient"

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
	gin.DisableConsoleColor()
	// Load configuration
	cfg := config.LoadConfig()
	logger.InitLogger("product-service", cfg.Environment)

	// Set up database connection
	db, err := infrastructure.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		logger.Log.Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	// Set up Consul
	consulClient, err := consulclient.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		logger.Log.Error("Failed to create Consul client", zap.Error(err))
		os.Exit(1)
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
		logger.Log.Error("Failed to initialize consumer group", zap.Error(err))
		os.Exit(1)
	}

	// Worker for consuming order messages
	orderWorker := worker.NewOrderWorker(redisBrokerClient, svc)
	go orderWorker.ListenForOrders(ctx)

	stockInsufficientWorker := worker.NewPaymentFailedWorker(redisBrokerClient, svc)
	go stockInsufficientWorker.ListenForPaymentFailures(ctx)

	r := gin.New()
	r.Use(sharedMiddleware.GinLogger())

	// Define Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

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
	grpcReady := make(chan struct{})
	var grpcServer *grpc.Server
	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			logger.Log.Error("Failed to listen on gRPC port", zap.String("port", cfg.GRPCPort), zap.Error(err))
			os.Exit(1)
		}

		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewProductGRPCServer(svc)
		pb.RegisterProductServiceServer(grpcServer, grpcHandler)

		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("product-service", grpc_health_v1.HealthCheckResponse_SERVING)

		logger.Log.Info("gRPC server starting", zap.String("port", cfg.GRPCPort))

		close(grpcReady) 

		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Error("gRPC server stopped", zap.Error(err))
		}
	}()

	<-grpcReady

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("product-service-%s", hostname)
	err = consulClient.RegisterService(serviceID, "product-service", "product-service", cfg.GRPCPort)
	if err != nil {
		logger.Log.Error("Failed to register service with Consul", zap.Error(err))
		os.Exit(1)
	}

	defer consulClient.DeregisterService(serviceID)

	// Start HTTP Server in a goroutine
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Product service starting", zap.String("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Failed to start HTTP server", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Log.Info("Shutting down servers")

	// Cancel context to stop worker
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("HTTP server forced to shutdown", zap.Error(err))
	}

	// Shutdown gRPC server
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	logger.Log.Info("Servers gracefully stopped")
}
