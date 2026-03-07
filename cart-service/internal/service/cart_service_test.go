package service

import (
	"context"
	"errors"
	"testing"

	"cart-service/internal/domain"
	"libs/pb"

	"google.golang.org/grpc"
)

type mockCartRepository struct {
	getCartItems []*domain.CartItem
	getCartErr   error

	savedItem   *domain.CartItem
	savedUserID string

	updatedUserID    string
	updatedProductID string
	updatedQty       uint
}

func (m *mockCartRepository) GetCart(ctx context.Context, userID string) ([]*domain.CartItem, error) {
	return m.getCartItems, m.getCartErr
}

func (m *mockCartRepository) GetCartItems(ctx context.Context, userID string, productIDs []uint) ([]*domain.CartItem, error) {
	return nil, nil
}

func (m *mockCartRepository) SaveCart(ctx context.Context, userID string, item *domain.CartItem) error {
	m.savedUserID = userID
	m.savedItem = item
	return nil
}

func (m *mockCartRepository) ClearCart(ctx context.Context, userID string) error { return nil }
func (m *mockCartRepository) DeleteCartItems(ctx context.Context, userID string, productIDs []uint) error {
	return nil
}

func (m *mockCartRepository) UpdateCartItem(ctx context.Context, userID string, productID string, qty uint) error {
	m.updatedUserID = userID
	m.updatedProductID = productID
	m.updatedQty = qty
	return nil
}

type mockProductClient struct {
	productResp *pb.ProductResponse
	productErr  error
}

func (m *mockProductClient) GetProduct(ctx context.Context, in *pb.GetProductRequest, opts ...grpc.CallOption) (*pb.ProductResponse, error) {
	if m.productErr != nil {
		return nil, m.productErr
	}
	return m.productResp, nil
}

func (m *mockProductClient) UpdateStock(ctx context.Context, in *pb.UpdateStockRequest, opts ...grpc.CallOption) (*pb.UpdateStockResponse, error) {
	return &pb.UpdateStockResponse{}, nil
}

func TestGetCartAggregatesTotals(t *testing.T) {
	repo := &mockCartRepository{getCartItems: []*domain.CartItem{{ProductID: 1, Quantity: 2, Price: 100}, {ProductID: 2, Quantity: 1, Price: 150}}}
	svc := NewCartService(repo, &mockProductClient{})

	cart, err := svc.GetCart(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetCart() error = %v", err)
	}
	if cart.TotalQty != 3 {
		t.Fatalf("expected total qty 3, got %d", cart.TotalQty)
	}
	if cart.TotalAmt != 350 {
		t.Fatalf("expected total amount 350, got %d", cart.TotalAmt)
	}
}

func TestAddToCartSavesNewItem(t *testing.T) {
	repo := &mockCartRepository{getCartItems: []*domain.CartItem{}}
	svc := NewCartService(repo, &mockProductClient{productResp: &pb.ProductResponse{Name: "Keyboard", Price: 500}})

	err := svc.AddToCart(context.Background(), 10, &domain.AddCartItemRequest{ProductID: 99, Quantity: 2})
	if err != nil {
		t.Fatalf("AddToCart() error = %v", err)
	}
	if repo.savedItem == nil {
		t.Fatal("expected SaveCart to be called")
	}
	if repo.savedItem.Name != "Keyboard" || repo.savedItem.Price != 500 {
		t.Fatalf("unexpected saved item: %#v", repo.savedItem)
	}
}

func TestAddToCartUpdatesExistingItemQuantity(t *testing.T) {
	repo := &mockCartRepository{getCartItems: []*domain.CartItem{{ProductID: 5, Quantity: 3, Price: 100}}}
	svc := NewCartService(repo, &mockProductClient{productResp: &pb.ProductResponse{Name: "Mouse", Price: 100}})

	err := svc.AddToCart(context.Background(), 21, &domain.AddCartItemRequest{ProductID: 5, Quantity: 2})
	if err != nil {
		t.Fatalf("AddToCart() error = %v", err)
	}
	if repo.updatedQty != 5 {
		t.Fatalf("expected updated qty 5, got %d", repo.updatedQty)
	}
	if repo.savedItem != nil {
		t.Fatal("did not expect SaveCart when item already exists")
	}
}

func TestAddToCartReturnsErrorWhenProductLookupFails(t *testing.T) {
	repo := &mockCartRepository{}
	svc := NewCartService(repo, &mockProductClient{productErr: errors.New("grpc unavailable")})

	err := svc.AddToCart(context.Background(), 1, &domain.AddCartItemRequest{ProductID: 2, Quantity: 1})
	if err == nil {
		t.Fatal("expected AddToCart to fail when product lookup fails")
	}
}
