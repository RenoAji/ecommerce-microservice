package main

import (
	"cart-service/internal/config"
	"cart-service/internal/handler"
	"cart-service/internal/infrastructure"
	"cart-service/internal/middleware"
	"cart-service/internal/repository"
	"cart-service/internal/service"
	"cart-service/internal/worker"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"libs/consulclient"
	"libs/logger"
	sharedMiddleware "libs/middleware/gin"
	"libs/pb"

	_ "cart-service/docs"
)

// @title Cart Service API
// @version 1.0
// @description This is the cart management service for the e-commerce platform.
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
	// Load configuration
	gin.DisableConsoleColor()
	cfg := config.LoadConfig()
	logger.InitLogger("cart-service", cfg.Environment)

	// Set up Redis client for cart storage
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	// Redis Broker Client for events
	redisBrokerClient := infrastructure.NewRedisBroker(
		cfg.GetRedisBrokerAddr(),
		cfg.RedisBroker.Password,
		cfg.RedisBroker.DB,
	)

	// Set up Consul
	consulClient, err := consulclient.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		logger.Log.Error("Failed to create Consul client", zap.Error(err))
		os.Exit(1)
	}

	// Set up gRPC connection to Product Service
	productClient := infrastructure.NewProductGRPCClient(cfg.ConsulAddr)

	repo := repository.NewRedisCartRepository(rdb)
	svc := service.NewCartService(repo, productClient)
	hdl := handler.NewCartHandler(svc)

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis Consumer Group
	err = infrastructure.InitCartConsumerGroup(ctx, redisBrokerClient)
	if err != nil {
		logger.Log.Error("Failed to initialize consumer group", zap.Error(err))
		os.Exit(1)
	}

	// Worker for consuming payment success messages
	paidSuccessWorker := worker.NewOrderPaidWorker(redisBrokerClient, svc)
	go paidSuccessWorker.Listen(ctx)

	// Worker for consuming payment failed messages
	paymentFailedWorker := worker.NewPaymentFailedWorker(redisBrokerClient, svc)
	go paymentFailedWorker.ListenForPaymentFailed(ctx)

	// Register routes
	r := gin.New()
	r.Use(sharedMiddleware.GinLogger())

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		cart := api.Group("/cart")
		cart.Use(middleware.AuthMiddleware()) // Apply auth middleware to all cart routes
		{
			// Define cart routes here, e.g.:
			cart.GET("", hdl.GetCart)
			cart.POST("/item", hdl.AddToCart)
			cart.PUT("/item/:product_id", hdl.UpdateCartItem)
			cart.DELETE("/item/:product_id", hdl.RemoveFromCart)
			cart.DELETE("", hdl.ClearCart)
		}
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
		grpcHandler := handler.NewCartGRPCServer(svc)
		pb.RegisterCartServiceServer(grpcServer, grpcHandler)

		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("cart-service", grpc_health_v1.HealthCheckResponse_SERVING)

		logger.Log.Info("gRPC server starting", zap.String("port", cfg.GRPCPort))

		close(grpcReady)

		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Error("gRPC server stopped", zap.Error(err))
		}
	}()

	<-grpcReady

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("cart-service-%s", hostname)
	err = consulClient.RegisterService(serviceID, "cart-service", "cart-service", cfg.GRPCPort)
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
		logger.Log.Info("Cart service starting", zap.String("port", cfg.ServerPort))
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
}
