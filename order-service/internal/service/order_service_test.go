package service

import (
	"context"
	"testing"

	"order-service/internal/domain"
	"order-service/pb"

	"google.golang.org/grpc"
)

type mockOrderRepo struct {
	orders             []domain.Order
	ordersErr          error
	updatedOrderID     string
	updatedStatus      string
	getOrderByIDResp   *domain.Order
	getOrderByIDErr    error
}

func (m *mockOrderRepo) AddOrder(ctx context.Context, order *domain.Order) error { return nil }
func (m *mockOrderRepo) GetOrders(ctx context.Context, userID uint, status string) ([]domain.Order, error) {
	return m.orders, m.ordersErr
}
func (m *mockOrderRepo) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return m.getOrderByIDResp, m.getOrderByIDErr
}
func (m *mockOrderRepo) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	m.updatedOrderID = orderID
	m.updatedStatus = status
	return nil
}
func (m *mockOrderRepo) UpdatePaymentUrl(ctx context.Context, orderID string, paymentURL string) error { return nil }

type mockOrderEventRepo struct {
	paidCalled  bool
	paidOrderID string
}

func (m *mockOrderEventRepo) PublishOrderCreatedEvent(ctx context.Context, event *domain.OrderEvent) error { return nil }
func (m *mockOrderEventRepo) PublishOrderPaidEvent(ctx context.Context, event *domain.OrderEvent) error {
	m.paidCalled = true
	m.paidOrderID = event.OrderID
	return nil
}

type mockOrderCartClient struct {
	userCartResp *pb.CartResponse
	userCartErr  error
}

func (m *mockOrderCartClient) GetUserCart(ctx context.Context, in *pb.GetCartRequest, opts ...grpc.CallOption) (*pb.CartResponse, error) {
	return m.userCartResp, m.userCartErr
}
func (m *mockOrderCartClient) ClearUserCart(ctx context.Context, in *pb.GetCartRequest, opts ...grpc.CallOption) (*pb.EmptyResponse, error) {
	return &pb.EmptyResponse{}, nil
}
func (m *mockOrderCartClient) GetCartItems(ctx context.Context, in *pb.GetCartItemRequest, opts ...grpc.CallOption) (*pb.CartResponse, error) {
	return &pb.CartResponse{}, nil
}
func (m *mockOrderCartClient) RemoveCartItems(ctx context.Context, in *pb.GetCartItemRequest, opts ...grpc.CallOption) (*pb.EmptyResponse, error) {
	return &pb.EmptyResponse{}, nil
}

type mockOrderProductClient struct{}

func (m *mockOrderProductClient) GetProduct(ctx context.Context, in *pb.GetProductRequest, opts ...grpc.CallOption) (*pb.ProductResponse, error) {
	return &pb.ProductResponse{}, nil
}
func (m *mockOrderProductClient) UpdateStock(ctx context.Context, in *pb.UpdateStockRequest, opts ...grpc.CallOption) (*pb.UpdateStockResponse, error) {
	return &pb.UpdateStockResponse{}, nil
}

type mockOrderPaymentClient struct{}

func (m *mockOrderPaymentClient) GetPaymentURL(ctx context.Context, in *pb.GetPaymentRequest, opts ...grpc.CallOption) (*pb.GetPaymentResponse, error) {
	return &pb.GetPaymentResponse{PaymentUrl: "https://example.com/pay"}, nil
}

func TestGetOrdersRejectsInvalidStatus(t *testing.T) {
	svc := NewOrderService(&mockOrderRepo{}, &mockOrderEventRepo{}, &mockOrderCartClient{}, &mockOrderProductClient{}, &mockOrderPaymentClient{})

	_, err := svc.GetOrders(context.Background(), 1, "NOT_A_STATUS")
	if err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestCreateOrderReturnsErrorWhenCartIsEmpty(t *testing.T) {
	svc := NewOrderService(
		&mockOrderRepo{},
		&mockOrderEventRepo{},
		&mockOrderCartClient{userCartResp: &pb.CartResponse{Items: []*pb.CartItem{}}},
		&mockOrderProductClient{},
		&mockOrderPaymentClient{},
	)

	_, err := svc.CreateOrder(&domain.CreateOrderRequest{}, context.Background(), 10)
	if err == nil {
		t.Fatal("expected CreateOrder to fail for empty cart")
	}
}

func TestUpdateOrderToPaidUpdatesRepoAndPublishesEvent(t *testing.T) {
	repo := &mockOrderRepo{getOrderByIDResp: &domain.Order{ID: 22, UserID: 7, TotalAmount: 900, Items: []domain.OrderItem{{ProductID: 1, Quantity: 2}}}}
	eventRepo := &mockOrderEventRepo{}
	svc := NewOrderService(repo, eventRepo, &mockOrderCartClient{}, &mockOrderProductClient{}, &mockOrderPaymentClient{})

	err := svc.UpdateOrderToPaid(context.Background(), "22")
	if err != nil {
		t.Fatalf("UpdateOrderToPaid() error = %v", err)
	}
	if repo.updatedOrderID != "22" || repo.updatedStatus != "PAID" {
		t.Fatalf("expected order 22 status PAID, got order=%s status=%s", repo.updatedOrderID, repo.updatedStatus)
	}
	if !eventRepo.paidCalled || eventRepo.paidOrderID != "22" {
		t.Fatalf("expected paid event for order 22, got called=%v order=%s", eventRepo.paidCalled, eventRepo.paidOrderID)
	}
}
