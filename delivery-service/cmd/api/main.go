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
	"libs/pb"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"

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
	// load configuration
	cfg := config.LoadConfig()

	// Setup postgres db
	db, err := infrastructure.InitializeDatabase(cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Set up Consul
	consulClient, err := consulclient.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		log.Fatal("Failed to create Consul client: ", err)
	}

	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("delivery-service-%s", hostname)
	err = consulClient.RegisterService(serviceID, "delivery-service", "delivery-service", cfg.GRPCPort)
	if err != nil {
		log.Fatal("Failed to register service with Consul: ", err)
	}
	defer consulClient.DeregisterService(serviceID)

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
		log.Fatalf("Failed to initialize consumer group: %v", err)
	}

	// Worker for consuming order paid messages
	orderPaidWorker := worker.NewOrderPaidWorker(redisBrokerClient, svc)
	go orderPaidWorker.Listen(ctx)

	// Outbox worker for publishing events
	outboxWorker := worker.NewOutboxWorker(svc)
	go outboxWorker.ListenForOutboxMessages(ctx)

	// define routes
	r := gin.Default()

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Use(middleware.AdminMiddleware())
	api := r.Group("/api/v1")
	{
		delivery := api.Group("/delivery")
		{
			delivery.GET("/", hdl.ListDeliveries)
			delivery.GET("/:id", hdl.GetDelivery)
			delivery.PUT("/:id/status", hdl.UpdateDeliveryStatus)
		}
		delivery.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	// Start gRPC Server in a goroutine
	var grpcServer *grpc.Server
	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port %s: %v", cfg.GRPCPort, err)
		}

		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewDeliveryGRPCServer(svc)
		pb.RegisterDeliveryServiceServer(grpcServer, grpcHandler)

		log.Println("gRPC server starting on port " + cfg.GRPCPort + "...")
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
		log.Println("Delivery Service starting on port " + cfg.ServerPort + "...")
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
