package service

import (
	"context"
	"fmt"
	"libs/logger"
	"product-service/internal/domain"
	"product-service/internal/repository"
	"strings"

	"go.uber.org/zap"
)

type ProductService struct {
	productRepo repository.ProductRepository
	eventRepo   repository.EventRepository
}

func NewProductService(pr repository.ProductRepository, er repository.EventRepository) *ProductService {
	return &ProductService{productRepo: pr, eventRepo: er}
}

func (s *ProductService) CreateProduct(ctx context.Context, product *domain.CreateProductRequest) error {
	l := logger.ForContext(ctx)
	err := s.productRepo.SaveProduct(product)
	if err != nil {
		l.Error("failed to create product", zap.Error(err))
		return fmt.Errorf("failed to create product: %w", err)
	}
	l.Info("Product created successfully", zap.String("name", product.Name), zap.Int("categoryCount", len(product.CategoryIDs)))
	return nil
}

func (s *ProductService) CreateCategory(ctx context.Context, category *domain.Category) error {
	l := logger.ForContext(ctx)
	err := s.productRepo.CreateCategory(category)
	if err != nil {
		l.Error("failed to create category", zap.Error(err))
		return fmt.Errorf("failed to create category: %w", err)
	}
	l.Info("Category created successfully", zap.String("name", category.Name))
	return nil
}

func (s *ProductService) GetProducts(ctx context.Context, search, categoryID, minPrice, maxPrice, order, sortBy string, page, limit int) (*domain.PaginatedProducts, error) {
	l := logger.ForContext(ctx)
	// Set default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	// Set max limit to prevent abuse
	if limit > 100 {
		limit = 100
	}

	products, total, err := s.productRepo.ListAll(search, categoryID, minPrice, maxPrice, order, sortBy, page, limit)
	if err != nil {
		l.Error("failed to list products", zap.Error(err))
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	productsResponse := make([]domain.ProductResponse, len(products))

	// Map each individual product
	for i, p := range products {
		productsResponse[i] = domain.ToProductResponse(p)
	}

	l.Info("Products retrieved successfully", zap.Int("count", len(productsResponse)), zap.Int("page", page), zap.Int("limit", limit))

	return &domain.PaginatedProducts{
		Products:   productsResponse,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ProductService) GetProductByID(ctx context.Context, productID uint) (*domain.Product, error) {
	l := logger.ForContext(ctx)
	product, err := s.productRepo.GetByID(productID)
	if err != nil {
		l.Error("failed to get product by id", zap.Error(err))
		return nil, fmt.Errorf("failed to get product by id: %w", err)
	}
	l.Info("Product retrieved successfully", zap.Uint("productID", productID))
	return product, nil
}

func (s *ProductService) AddStock(ctx context.Context, productID uint, add int) error {
	l := logger.ForContext(ctx)
	err := s.productRepo.AddStock(productID, add)
	if err != nil {
		l.Error("failed to add stock", zap.Error(err))
		return fmt.Errorf("failed to add stock: %w", err)
	}
	l.Info("Product stock added successfully", zap.Uint("productID", productID), zap.Int("added", add))
	return nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, productID uint) error {
	l := logger.ForContext(ctx)
	err := s.productRepo.Delete(productID)
	if err != nil {
		l.Error("failed to delete product", zap.Error(err))
		return fmt.Errorf("failed to delete product: %w", err)
	}
	l.Info("Product deleted successfully", zap.Uint("productID", productID))
	return nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, id uint, product *domain.UpdateProductRequest) (*domain.Product, error) {
	l := logger.ForContext(ctx)
	updatedProduct, err := s.productRepo.UpdateProduct(id, product)
	if err != nil {
		l.Error("failed to update product", zap.Error(err))
		return nil, fmt.Errorf("failed to update product: %w", err)
	}
	l.Info("Product updated successfully", zap.Uint("productID", id))
	return updatedProduct, nil
}

func (s *ProductService) ReserveStock(ctx context.Context, orderID uint, stockUpdates map[uint]int) error {
	l := logger.ForContext(ctx)
	// Deduct stocks in a transaction
	err := s.productRepo.AddStocksInTransaction(stockUpdates)
	if err != nil {
		// Check if error is due to insufficient stock
		if strings.Contains(err.Error(), "resulting stock would be negative") {
			// Publish stock insufficient event
			publishErr := s.eventRepo.PublishStockInsufficientEvent(ctx, &domain.StockEvent{
				OrderID:       orderID,
				CorrelationID: correlationIDFromContext(ctx),
			})
			if publishErr != nil {
				l.Error("failed to reserve stock and publish stock insufficient event: reserveErr=%w, publishErr=%v", zap.Error(err), zap.Error(publishErr))
				return fmt.Errorf("failed to reserve stock and publish stock insufficient event: reserveErr=%w, publishErr=%v", err, publishErr)
			}
			l.Info("Stock insufficient event published", zap.Uint("orderID", orderID))
		}

		l.Error("failed to reserve stock for order %d", zap.Error(err))
		return fmt.Errorf("failed to reserve stock for order %d: %w", orderID, err)
	}

	// Publish event after successful stock deduction
	err = s.eventRepo.PublishStockReservedEvent(ctx, &domain.StockEvent{
		OrderID:       orderID,
		CorrelationID: correlationIDFromContext(ctx),
	})
	if err != nil {
		l.Error("failed to publish stock reserved event for order %d", zap.Error(err))
		return fmt.Errorf("failed to publish stock reserved event for order %d: %w", orderID, err)
	}
	l.Info("Stock reserved successfully", zap.Uint("orderID", orderID))

	return nil
}

func (s *ProductService) ReleaseStock(ctx context.Context, stockUpdates map[uint]int) error {
	l := logger.ForContext(ctx)
	// Add stocks back in a transaction
	err := s.productRepo.AddStocksInTransaction(stockUpdates)
	if err != nil {
		l.Error("failed to release stock", zap.Error(err))
		return fmt.Errorf("failed to release stock: %w", err)
	}
	l.Info("Stock released successfully", zap.Int("itemCount", len(stockUpdates)))

	return nil
}

func correlationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}

	return ""
}
