package main

import (
	"context"
	"log"
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
	"payment-service/pb"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"

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
	// load configuration
	cfg := config.LoadConfig()

	// DB
	db, err := infrastructure.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Auto-migrate (creates the table if it doesn't exist)
	db.AutoMigrate(&domain.Payment{})


	// Midtrans Client
	midtransClient := infrastructure.NewMidtransClient(cfg.MidtransServerKey, cfg.MidtransClientKey)

	// Redis Client for broker
	redisClient := infrastructure.NewRedisBroker(cfg.GetRedisAddr(), cfg.RedisBroker.Password, cfg.RedisBroker.DB)
	
	// Repository, Service, Handler setup
	repo := repository.NewPostgresRepository(db)
	eventRepo := repository.NewRedisRepository(redisClient)
	svc := service.NewPaymentService(repo,eventRepo, midtransClient)
	hdl := handler.NewPaymentHandler(svc)

	// GRPC Server
	var grpcServer *grpc.Server
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Failed to listen on port 50051: %v", err)
		}
		
		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewPaymentGRPCServer(svc)
		pb.RegisterPaymentServiceServer(grpcServer, grpcHandler)
		
		log.Println("gRPC server starting on port 50051...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	// register routes
	r:= gin.Default()

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api:= r.Group("/api/v1")
	{
		payment:= api.Group("/payment")
		payment.POST("/webhook", hdl.HandleWebhook)
	}

	// Start HTTP Server in a goroutine
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}
	
	go func() {
		log.Println("Payment Service starting on port " + cfg.ServerPort + "...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
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
	log.Println("Shutting down servers...")

	// Cancel the context to stop the cleanup worker
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