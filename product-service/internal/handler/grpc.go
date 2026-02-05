package handler

import (
	"context"
	"product-service/internal/service"
	"product-service/pb" // The folder where your generated code lives

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductGRPCServer struct {
    pb.UnimplementedProductServiceServer
    service *service.ProductService
}

func NewProductGRPCServer(service *service.ProductService) *ProductGRPCServer {
    return &ProductGRPCServer{service: service}
}
func (s *ProductGRPCServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
    // 1. Call your existing business logic
    id := uint64(req.Id)
    p, err := s.service.GetProductByID(uint(id))
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "product not found")
    }
    

    // 2. Map domain entity to Protobuf response
    return &pb.ProductResponse{
        Id:    uint32(p.ID),
        Name:  p.Name,
        Price: uint64(p.Price),
    }, nil
}

func (s *ProductGRPCServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
    // 1. Call your existing business logic
    id := uint(req.Id)
    
    err := s.service.AddStock(id, int(req.Add))
    if err != nil {
        return nil, status.Errorf(codes.Internal, "could not update stock")
    }

    // 2. Return success response
    return &pb.UpdateStockResponse{
        Message: "Stock updated successfully",
    }, nil
}