package infrastructure

import (
	"fmt"

	"cart-service/pb"
	"libs/infrastructure"
)

func NewProductGRPCClient(address string) pb.ProductServiceClient {
	target := fmt.Sprintf("consul://%s/product-service?wait=14s", address)
	conn := infrastructure.NewGRPCClient(target)
	return pb.NewProductServiceClient(conn)
}
