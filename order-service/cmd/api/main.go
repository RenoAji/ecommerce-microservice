package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"order-service/internal/config"
	"order-service/internal/domain"
	"order-service/internal/handler"
	"order-service/internal/infrastructure"
	"order-service/internal/middleware"
	"order-service/internal/repository"
	"order-service/internal/service"
	"order-service/internal/worker"

	"libs/consulclient"
	"libs/logger"
	sharedMiddleware "libs/middleware/gin"

	_ "order-service/docs"
	// _ "github.com/mbobakov/grpc-consul-resolver"
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
	gin.DisableConsoleColor()
	// Load configuration
	cfg := config.LoadConfig()
	logger.InitLogger("order-service", cfg.Environment)

	// Set up database connection
	db, err := infrastructure.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		logger.Log.Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	// Auto-migrate (creates the table if it doesn't exist)
	db.AutoMigrate(&domain.Order{}, &domain.OrderItem{})

	// Set up Consul
	consulClient, err := consulclient.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		logger.Log.Error("Failed to create Consul client", zap.Error(err))
		os.Exit(1)
	}

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("order-service-%s", hostname)
	err = consulClient.RegisterService(serviceID, "order-service", "order-service", cfg.GRPCPort)
	if err != nil {
		logger.Log.Error("Failed to register service with Consul", zap.Error(err))
		os.Exit(1)
	}
	defer consulClient.DeregisterService(serviceID)

	// Set up gRPC connection to Cart Service
	CartClient := infrastructure.NewCartGRPCClient(cfg.ConsulAddr)

	// Set up gRPC connection to Product Service
	ProductClient := infrastructure.NewProductGRPCClient(cfg.ConsulAddr)

	// Set up GRPC connection to Payment Service
	PaymentClient := infrastructure.NewPaymentGRPCClient(cfg.ConsulAddr)

	// Redis Broker Client
	redisBrokerClient := infrastructure.NewRedisBroker(cfg.GetRedisAddr(), cfg.RedisBroker.Password, cfg.RedisBroker.DB)

	// Initialize repositories, services, and handlers
	repo := repository.NewPostgresRepository(db)
	brokerRepo := repository.NewRedisRepository(redisBrokerClient)
	svc := service.NewOrderService(repo, brokerRepo, CartClient, ProductClient, PaymentClient)
	hdl := handler.NewOrderHandler(svc)

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis Consumer Group
	err = infrastructure.InitOrderConsumerGroup(ctx, redisBrokerClient)
	if err != nil {
		logger.Log.Error("Failed to initialize consumer group", zap.Error(err))
		os.Exit(1)
	}

	// Worker for consuming order messages
	StockreservedWorker := worker.NewStockreservedWorker(redisBrokerClient, svc)
	go StockreservedWorker.ListenForStockReserved(ctx)

	// Worker for consuming stock insufficient messages
	StockInsufficientWorker := worker.NewStockInsufficientWorker(redisBrokerClient, svc)
	go StockInsufficientWorker.ListenForStockInsufficient(ctx)

	// Worker for consuming payment success messages
	PaidSuccessWorker := worker.NewPaidSuccessWorker(redisBrokerClient, svc)
	go PaidSuccessWorker.ListenForPaidSuccess(ctx)

	// Worker for consuming payment failed messages
	PaymentFailedWorker := worker.NewPaymentFailedWorker(redisBrokerClient, svc)
	go PaymentFailedWorker.ListenForPaymentFailed(ctx)

	// Worker for consuming delivery success messages
	DeliverySuccessWorker := worker.NewDeliverySuccessWorker(redisBrokerClient, svc)
	go DeliverySuccessWorker.ListenForDeliverySuccess(ctx)

	// Worker for consuming delivery failed messages
	DeliveryFailedWorker := worker.NewDeliveryFailedWorker(redisBrokerClient, svc)
	go DeliveryFailedWorker.ListenForDeliveryFailed(ctx)

	// register routes
	r := gin.New()
	r.Use(sharedMiddleware.GinLogger())

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		order := api.Group("/order")
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
	// log.Println("Order Service starting on port 8081...")
	// if err := r.Run(":8081"); err != nil {
	// 	log.Fatal("Failed to start server: ", err)
	// }

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Order service starting", zap.String("port", cfg.ServerPort))
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
}
