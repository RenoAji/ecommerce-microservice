package handler

import (
	"cart-service/internal/service"
	"cart-service/pb"
	"context"
	"strconv"
)

type CartGRPCServer struct {
    pb.UnimplementedCartServiceServer
    service *service.CartService
}

func NewCartGRPCServer(service *service.CartService) *CartGRPCServer {
    return &CartGRPCServer{service: service}
}

func (s *CartGRPCServer) GetUserCart(ctx context.Context, req *pb.GetCartRequest) (*pb.CartResponse, error) {
	userId, err := strconv.ParseUint(req.UserId, 10, 64)
	cart, err := s.service.GetCart(ctx, uint(userId))
	if err != nil {
		return nil, err
	}

	// Pre-allocate memory to avoid multiple re-allocations
	items := make([]*pb.CartItem, 0, len(cart.Items)) 

	for _, item := range cart.Items {
		items = append(items, &pb.CartItem{
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			Price:     item.Price,
		})
	}

	return &pb.CartResponse{
		UserId:    cart.UserID,
		Items:     items,
		TotalPrice:  int64(cart.TotalAmt),
	}, nil
}

func (s *CartGRPCServer) GetCartItems(ctx context.Context, req *pb.GetCartItemRequest) (*pb.CartResponse, error) {
	userId, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		return nil, err
	}

	cart, err := s.service.GetCartItems(ctx, uint(userId), req.ProductIds)

	if err != nil {
		return nil, err
	}

	// Pre-allocate memory to avoid multiple re-allocations
	items := make([]*pb.CartItem, 0, len(cart)) 

	var totalAmt int64 = 0
	for _, item := range cart {
		items = append(items, &pb.CartItem{
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			Price:     item.Price,
		})
		totalAmt += int64(item.Quantity) * item.Price
	}

	return &pb.CartResponse{
		UserId:    req.UserId,
		Items:     items,
		TotalPrice:  totalAmt,
	}, nil
}

func (s *CartGRPCServer) RemoveCartItems(ctx context.Context, req *pb.GetCartItemRequest) (*pb.EmptyResponse, error) {
	userId, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		return nil, err
	}

	err = s.service.RemoveCartItems(ctx, uint(userId), req.ProductIds)
	if err != nil {
		return nil, err
	}

	return &pb.EmptyResponse{}, nil
}

func (s *CartGRPCServer) ClearUserCart(ctx context.Context, req *pb.GetCartRequest) (*pb.EmptyResponse, error) {
	userId, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		return nil, err
	}

	err = s.service.ClearCart(ctx, uint(userId))
	if err != nil {
		return nil, err
	}

	return &pb.EmptyResponse{}, nil
}