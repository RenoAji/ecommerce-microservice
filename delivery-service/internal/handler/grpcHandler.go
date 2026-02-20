package handler

import (
	"context"
	"delivery-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"delivery-service/pb"
)

type DeliveryGRPCServer struct {
	pb.UnimplementedDeliveryServiceServer
	service *service.DeliveryService
}

func NewDeliveryGRPCServer(service *service.DeliveryService) *DeliveryGRPCServer {
	return &DeliveryGRPCServer{service: service}
}

func (s *DeliveryGRPCServer) GetDeliveryByOrderId(ctx context.Context, req *pb.GetDeliveryByOrderIdRequest) (*pb.DeliveryResponse, error) {
	orderID := uint(req.OrderId)
	delivery, err := s.service.GetDeliveryByOrderID(orderID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "delivery not found")
	}

	return &pb.DeliveryResponse{
		Delivery: &pb.DeliveryInfo{
			OrderId: uint32(delivery.OrderID),
			Status:  delivery.Status,
		},
	}, nil
}