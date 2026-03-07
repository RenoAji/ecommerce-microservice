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
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"

	"libs/consulclient"
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
	cfg := config.LoadConfig()

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
		log.Fatal("Failed to create Consul client: ", err)
	}

	port, err := strconv.Atoi(cfg.ServerPort)
	if err != nil {
		log.Fatal("Failed to convert ServerPort to int: ", err)
	}
	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("cart-service-%s", hostname)
	consulClient.RegisterService(serviceID, "cart-service", port)
	defer consulClient.DeregisterService(serviceID)

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
		log.Fatalf("Failed to initialize consumer group: %v", err)
	}

	// Worker for consuming payment success messages
	paidSuccessWorker := worker.NewOrderPaidWorker(redisBrokerClient, svc)
	go paidSuccessWorker.Listen(ctx)

	// Worker for consuming payment failed messages
	paymentFailedWorker := worker.NewPaymentFailedWorker(redisBrokerClient, svc)
	go paymentFailedWorker.ListenForPaymentFailed(ctx)

	// Register routes
	r := gin.Default()

	api := r.Group("/api/v1")
	{
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
	var grpcServer *grpc.Server
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Failed to listen on port 50051: %v", err)
		}

		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewCartGRPCServer(svc)
		pb.RegisterCartServiceServer(grpcServer, grpcHandler)

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
		log.Println("Cart Service starting on port " + cfg.ServerPort + "...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down servers...")

	// Cancel context to stop workers
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
