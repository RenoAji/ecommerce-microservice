package service

import (
	"context"
	"errors"
	"testing"

	"product-service/internal/domain"
)

type mockProductRepository struct {
	listAllArgs struct {
		search     string
		categoryID string
		minPrice   string
		maxPrice   string
		order      string
		sortBy     string
		page       int
		limit      int
	}
	listAllProducts []domain.Product
	listAllTotal    int64
	listAllErr      error
	addStocksErr    error
}

func (m *mockProductRepository) SaveProduct(product *domain.CreateProductRequest) error { return nil }
func (m *mockProductRepository) CreateCategory(category *domain.Category) error         { return nil }
func (m *mockProductRepository) AddStock(productID uint, add int) error                 { return nil }
func (m *mockProductRepository) Delete(productID uint) error                            { return nil }
func (m *mockProductRepository) GetByID(productID uint) (*domain.Product, error)        { return nil, nil }
func (m *mockProductRepository) AssignCategory(productID uint, categoryID []uint) error { return nil }
func (m *mockProductRepository) RemoveCategory(productID uint, categoryID uint) error   { return nil }
func (m *mockProductRepository) ListCategories(productID uint) ([]domain.Category, error) {
	return nil, nil
}
func (m *mockProductRepository) UpdateProduct(id uint, req *domain.UpdateProductRequest) (*domain.Product, error) {
	return nil, nil
}
func (m *mockProductRepository) AddStocksInTransaction(updates map[uint]int) error {
	return m.addStocksErr
}
func (m *mockProductRepository) ListAll(search, categoryID, minPrice, maxPrice, order, sortBy string, page, limit int) ([]domain.Product, int64, error) {
	m.listAllArgs.search = search
	m.listAllArgs.categoryID = categoryID
	m.listAllArgs.minPrice = minPrice
	m.listAllArgs.maxPrice = maxPrice
	m.listAllArgs.order = order
	m.listAllArgs.sortBy = sortBy
	m.listAllArgs.page = page
	m.listAllArgs.limit = limit
	return m.listAllProducts, m.listAllTotal, m.listAllErr
}

type mockProductEventRepository struct {
	reservedCalled      bool
	insufficientCalled  bool
	reservedOrderID     uint
	insufficientOrderID uint
}

func (m *mockProductEventRepository) PublishStockReservedEvent(ctx context.Context, event *domain.StockEvent) error {
	m.reservedCalled = true
	m.reservedOrderID = event.OrderID
	return nil
}

func (m *mockProductEventRepository) PublishStockInsufficientEvent(ctx context.Context, event *domain.StockEvent) error {
	m.insufficientCalled = true
	m.insufficientOrderID = event.OrderID
	return nil
}

func TestGetProductsAppliesDefaultPagination(t *testing.T) {
	repo := &mockProductRepository{listAllProducts: []domain.Product{}, listAllTotal: 0}
	eventRepo := &mockProductEventRepository{}
	svc := NewProductService(repo, eventRepo)

	_, err := svc.GetProducts(context.Background(), "", "", "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetProducts() error = %v", err)
	}

	if repo.listAllArgs.page != 1 {
		t.Fatalf("expected page=1, got %d", repo.listAllArgs.page)
	}
	if repo.listAllArgs.limit != 10 {
		t.Fatalf("expected limit=10, got %d", repo.listAllArgs.limit)
	}
}

func TestReserveStockPublishesInsufficientEventWhenStockError(t *testing.T) {
	repo := &mockProductRepository{addStocksErr: errors.New("resulting stock would be negative")}
	eventRepo := &mockProductEventRepository{}
	svc := NewProductService(repo, eventRepo)

	err := svc.ReserveStock(context.Background(), 44, map[uint]int{1: -10})
	if err == nil {
		t.Fatal("expected reserve stock error")
	}
	if !eventRepo.insufficientCalled {
		t.Fatal("expected insufficient event to be published")
	}
	if eventRepo.insufficientOrderID != 44 {
		t.Fatalf("expected insufficient event orderID=44, got %d", eventRepo.insufficientOrderID)
	}
	if eventRepo.reservedCalled {
		t.Fatal("did not expect reserved event when stock reserve fails")
	}
}

func TestReserveStockPublishesReservedEventOnSuccess(t *testing.T) {
	repo := &mockProductRepository{}
	eventRepo := &mockProductEventRepository{}
	svc := NewProductService(repo, eventRepo)

	err := svc.ReserveStock(context.Background(), 55, map[uint]int{1: -2})
	if err != nil {
		t.Fatalf("ReserveStock() error = %v", err)
	}
	if !eventRepo.reservedCalled {
		t.Fatal("expected reserved event to be published")
	}
	if eventRepo.reservedOrderID != 55 {
		t.Fatalf("expected reserved event orderID=55, got %d", eventRepo.reservedOrderID)
	}
}
