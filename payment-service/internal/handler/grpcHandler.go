package handler

import (
	"context"
	"libs/pb"
	"payment-service/internal/service"
)

type PaymentGRPCServer struct {
	pb.UnimplementedPaymentServiceServer
	service *service.PaymentService
}

func NewPaymentGRPCServer(service *service.PaymentService) *PaymentGRPCServer {
	return &PaymentGRPCServer{service: service}
}

func (s *PaymentGRPCServer) GetPaymentURL(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
	url, err := s.service.CreatePendingPayment(ctx, uint(req.OrderId), uint(req.Amount))

	return &pb.GetPaymentResponse{
		PaymentUrl: url,
	}, err
}
