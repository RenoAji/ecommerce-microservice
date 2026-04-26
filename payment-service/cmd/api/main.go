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
	"payment-service/internal/config"
	"payment-service/internal/domain"
	"payment-service/internal/handler"
	"payment-service/internal/infrastructure"
	"payment-service/internal/middleware"
	"payment-service/internal/repository"
	"payment-service/internal/service"
	"payment-service/internal/worker"
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

	_ "payment-service/docs"
)

// @title Payment Service API
// @version 1.0
// @description Payment service for handling Midtrans payment webhooks and payment status
// @termsOfService http://swagger.io/terms/

// @contact.name Septareno Nugroho Aji
// @contact.email renoaji25sep@gmail.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8081
// @BasePath /api/v1

func main() {
	gin.DisableConsoleColor()
	// load configuration
	cfg := config.LoadConfig()
	logger.InitLogger("payment-service", cfg.Environment)

	// DB
	db, err := infrastructure.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		logger.Log.Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	// Auto-migrate (creates the table if it doesn't exist)
	db.AutoMigrate(&domain.Payment{})

	// Midtrans Client
	midtransClient := infrastructure.NewMidtransClient(cfg.MidtransServerKey, cfg.MidtransClientKey)

	// Set up Consul
	consulClient, err := consulclient.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		logger.Log.Error("Failed to create Consul client", zap.Error(err))
		os.Exit(1)
	}

	// Redis Client for broker
	redisClient := infrastructure.NewRedisBroker(cfg.GetRedisAddr(), cfg.RedisBroker.Password, cfg.RedisBroker.DB)

	// Repository, Service, Handler setup
	repo := repository.NewPostgresRepository(db)
	eventRepo := repository.NewRedisRepository(redisClient)
	svc := service.NewPaymentService(repo, eventRepo, midtransClient)
	hdl := handler.NewPaymentHandler(svc)

	// GRPC Server
	grpcReady := make(chan struct{})
	var grpcServer *grpc.Server
	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			logger.Log.Error("Failed to listen on gRPC port", zap.String("port", cfg.GRPCPort), zap.Error(err))
			os.Exit(1)
		}

		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewPaymentGRPCServer(svc)
		pb.RegisterPaymentServiceServer(grpcServer, grpcHandler)

		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("payment-service", grpc_health_v1.HealthCheckResponse_SERVING)

		logger.Log.Info("gRPC server starting", zap.String("port", cfg.GRPCPort))

		close(grpcReady)

		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Error("gRPC server stopped", zap.Error(err))
		}
	}()

	<-grpcReady

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("payment-service-%s", hostname)
	err = consulClient.RegisterService(serviceID, "payment-service", "payment-service", cfg.GRPCPort)
	if err != nil {
		logger.Log.Error("Failed to register service with Consul", zap.Error(err))
		os.Exit(1)
	}
	defer consulClient.DeregisterService(serviceID)

	// register routes
	r := gin.New()
	r.Use(sharedMiddleware.GinLogger())

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		payment := api.Group("/payment")
		payment.POST("/webhook", hdl.HandleWebhook)
	}

	// Start HTTP Server in a goroutine
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Payment service starting", zap.String("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Failed to start HTTP server", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Start cleanup worker for handling expired payments
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupWorker := worker.NewCleanupWorker(svc)
	go cleanupWorker.StartCleanupJob(ctx)

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Log.Info("Shutting down servers")

	// Cancel the context to stop the cleanup worker
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
