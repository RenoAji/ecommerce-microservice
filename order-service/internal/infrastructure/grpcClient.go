package infrastructure

import (
	"fmt"
	"libs/pb"

	"libs/infrastructure"
)

func NewCartGRPCClient(address string) pb.CartServiceClient {
	target := fmt.Sprintf("consul://%s/cart-service?wait=14s", address)
	conn := infrastructure.NewGRPCClient(target)
	return pb.NewCartServiceClient(conn)
}

func NewProductGRPCClient(address string) pb.ProductServiceClient {
	target := fmt.Sprintf("consul://%s/product-service?wait=14s", address)
	conn := infrastructure.NewGRPCClient(target)
	return pb.NewProductServiceClient(conn)
}

func NewPaymentGRPCClient(address string) pb.PaymentServiceClient {
	target := fmt.Sprintf("consul://%s/payment-service?wait=14s", address)
	conn := infrastructure.NewGRPCClient(target)
	return pb.NewPaymentServiceClient(conn)
}
