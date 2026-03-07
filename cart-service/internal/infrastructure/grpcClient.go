package infrastructure

import (
	"fmt"

	"libs/infrastructure"
	"libs/pb"
)

func NewProductGRPCClient(address string) pb.ProductServiceClient {
	target := fmt.Sprintf("consul://%s/product-service?wait=14s", address)
	conn := infrastructure.NewGRPCClient(target)
	return pb.NewProductServiceClient(conn)
}
