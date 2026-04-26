package main

import (
	"context"
	"delivery-service/internal/config"
	"delivery-service/internal/domain"
	"delivery-service/internal/handler"
	"delivery-service/internal/infrastructure"
	"delivery-service/internal/middleware"
	"delivery-service/internal/repository"
	"delivery-service/internal/service"
	"delivery-service/internal/worker"
	"fmt"
	"libs/logger"
	sharedMiddleware "libs/middleware/gin"
	"libs/pb"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"libs/consulclient"

	_ "delivery-service/docs"
)

// @title Delivery Service API
// @version 1.0
// @description This is the delivery management service for the e-commerce platform.
// @termsOfService http://swagger.io/terms/

// @contact.name Septareno Nugroho Aji
// @contact.email septareno@example.com

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
	// load configuration
	cfg := config.LoadConfig()
	logger.InitLogger("delivery-service", cfg.Environment)

	// Setup postgres db
	db, err := infrastructure.InitializeDatabase(cfg.GetDSN())
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
	db.AutoMigrate(&domain.Delivery{}, &domain.DeliveryOutboxMessage{})

	// Redis Broker Client for events
	redisBrokerClient := infrastructure.NewRedisBroker(
		cfg.GetRedisAddr(),
		cfg.RedisBroker.Password,
		cfg.RedisBroker.DB,
	)

	// handler, service, repo
	repo := repository.NewPostgresRepository(db)
	eventRepo := repository.NewRedisStreamRepository(redisBrokerClient)
	svc := service.NewDeliveryService(repo, eventRepo)
	hdl := handler.NewDeliveryHandler(svc)

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis Consumer Group
	err = infrastructure.InitDeliveryConsumerGroup(ctx, redisBrokerClient)
	if err != nil {
		logger.Log.Error("Failed to initialize consumer group", zap.Error(err))
		os.Exit(1)
	}

	// Worker for consuming order paid messages
	orderPaidWorker := worker.NewOrderPaidWorker(redisBrokerClient, svc)
	go orderPaidWorker.Listen(ctx)

	// Outbox worker for publishing events
	outboxWorker := worker.NewOutboxWorker(svc)
	go outboxWorker.ListenForOutboxMessages(ctx)

	// define routes
	r := gin.New()
	r.Use(sharedMiddleware.GinLogger())

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		delivery := api.Group("/delivery")
		delivery.Use(middleware.AdminMiddleware())
		{
			delivery.GET("/", hdl.ListDeliveries)
			delivery.GET("/:id", hdl.GetDelivery)
			delivery.PUT("/:id/status", hdl.UpdateDeliveryStatus)
		}
	}

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
		grpcHandler := handler.NewDeliveryGRPCServer(svc)
		pb.RegisterDeliveryServiceServer(grpcServer, grpcHandler)

		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("delivery-service", grpc_health_v1.HealthCheckResponse_SERVING)

		logger.Log.Info("gRPC server starting", zap.String("port", cfg.GRPCPort))

		close(grpcReady)

		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Error("gRPC server stopped", zap.Error(err))
		}
	}()

	<-grpcReady

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("delivery-service-%s", hostname)
	err = consulClient.RegisterService(serviceID, "delivery-service", "delivery-service", cfg.GRPCPort)
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
		logger.Log.Info("Delivery service starting", zap.String("port", cfg.ServerPort))
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

	// Cancel context to stop workers
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
	defer logger.Log.Sync()
}
