package main

import (
	"cart-service/internal/handler"
	"cart-service/internal/middleware"
	"cart-service/internal/repository"
	"cart-service/internal/service"
	"cart-service/pb"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"os"

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
	host:= os.Getenv("REDIS_HOST")
	port:= os.Getenv("REDIS_PORT")
	password:= os.Getenv("REDIS_PASSWORD")
	
	rdb := redis.NewClient(&redis.Options{
        Addr:     host + ":" + port,
        Password: password,
        DB:       0,
    })
    defer rdb.Close()

	// Set up gRPC connection to Product Service
	conn, err := grpc.NewClient("product-service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    
    // Create the gRPC client stub
    productClient := pb.NewProductServiceClient(conn)

	repo:= repository.NewRedisCartRepository(rdb)
	svc:= service.NewCartService(repo, productClient)
	hdl:= handler.NewCartHandler(svc)

	// Register routes
	r := gin.Default()

	api:= r.Group("/api/v1")
	{
		cart:= api.Group("/cart")
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

		// Start gRPC Server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Failed to listen on port 50051: %v", err)
		}
		
		grpcServer := grpc.NewServer(grpc.UnaryInterceptor(middleware.InternalAuthInterceptor))
		grpcHandler := handler.NewCartGRPCServer(svc)
		pb.RegisterCartServiceServer(grpcServer, grpcHandler)
		
		log.Println("gRPC server starting on port 50051...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Swagger Documentation Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8081") // Start server on port 8081
}