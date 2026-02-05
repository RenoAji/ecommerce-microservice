package infrastructure

import (
	"log"
	"order-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewGRPCClient(address string) (*grpc.ClientConn) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
	return conn
}

func NewCartGRPCClient(address string) pb.CartServiceClient {
	conn := NewGRPCClient(address)
	return pb.NewCartServiceClient(conn)
}

func NewProductGRPCClient(address string) pb.ProductServiceClient {
	conn := NewGRPCClient(address)
	return pb.NewProductServiceClient(conn)
}

func NewPaymentGRPCClient(address string) pb.PaymentServiceClient {
	conn := NewGRPCClient(address)
	return pb.NewPaymentServiceClient(conn)
}