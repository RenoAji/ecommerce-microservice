package handler

import (
	"context"
	"payment-service/internal/service"
	"payment-service/pb"
)

type PaymentGRPCServer struct {
    pb.UnimplementedPaymentServiceServer
    service *service.PaymentService
}

func NewPaymentGRPCServer(service *service.PaymentService) *PaymentGRPCServer {
    return &PaymentGRPCServer{service: service}
}

func (s *PaymentGRPCServer) GetPaymentURL(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
    url, err := s.service.CreatePendingPayment(uint(req.OrderId), uint(req.Amount))

    return &pb.GetPaymentResponse{
        PaymentUrl: url,
    }, err
}