package infrastructure

import (
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/mbobakov/grpc-consul-resolver"
)

func NewGRPCClient(address string) (*grpc.ClientConn) {
	conn, err := grpc.NewClient(
		address, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
	)
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
	return conn
}