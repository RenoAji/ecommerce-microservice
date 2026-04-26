package infrastructure

import (
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/mbobakov/grpc-consul-resolver"
)

func NewGRPCClient(address string) (*grpc.ClientConn) {
	internalSecret := os.Getenv("INTERNAL_SECRET")
	clientInterceptor := newClientInterceptors(internalSecret)
	
	conn, err := grpc.NewClient(
		address, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithUnaryInterceptor(clientInterceptor),
	)
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
	return conn
}