package service

import (
	"context"
	"log"
	"product-service/internal/domain"
	"product-service/internal/repository"
	"strings"
)

type ProductService struct {
	productRepo repository.ProductRepository
	eventRepo   repository.EventRepository
}

func NewProductService(pr repository.ProductRepository, er repository.EventRepository) *ProductService {
	return &ProductService{productRepo: pr, eventRepo: er}
}

func (s *ProductService) CreateProduct(product *domain.CreateProductRequest) error {
	return s.productRepo.SaveProduct(product)
}

func (s *ProductService) CreateCategory(category *domain.Category) error {
	return s.productRepo.CreateCategory(category)
}



func (s *ProductService) GetProducts(search, categoryID, minPrice, maxPrice, order, sortBy string, page, limit int) (*domain.PaginatedProducts, error) {
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
		return nil, err
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

	return &domain.PaginatedProducts{
		Products:   productsResponse,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ProductService) GetProductByID(productID uint) (*domain.Product, error) {
	return s.productRepo.GetByID(productID)
}

func (s *ProductService) AddStock(productID uint, add int) error {
	return s.productRepo.AddStock(productID, add)
}

func (s *ProductService) DeleteProduct(productID uint) error {
	return s.productRepo.Delete(productID)
}

func (s *ProductService) UpdateProduct(id uint, product *domain.UpdateProductRequest) (*domain.Product, error) {
	return s.productRepo.UpdateProduct(id, product)
}

func (s *ProductService) ReserveStock(ctx context.Context, orderID uint, stockUpdates map[uint]int) error {
	// Deduct stocks in a transaction
	err := s.productRepo.AddStocksInTransaction(stockUpdates)
	if err != nil {
		log.Printf("Failed to reserve stock for order %d: %v", orderID, err)
		
		// Check if error is due to insufficient stock
		if strings.Contains(err.Error(), "resulting stock would be negative") {
			// Publish stock insufficient event
			publishErr := s.eventRepo.PublishStockInsufficientEvent(ctx, &domain.StockEvent{
				OrderID: orderID,
			})
			if publishErr != nil {
				log.Printf("Failed to publish stock insufficient event for order %d: %v", orderID, publishErr)
			}
		}
		
		return err
	}

	// Publish event after successful stock deduction
	err = s.eventRepo.PublishStockReservedEvent(ctx ,&domain.StockEvent{
		OrderID: orderID,
	})
	if err != nil {
		log.Printf("Failed to publish stock reserved event for order %d: %v", orderID, err)
		return err
	}

	return nil
}

func (s *ProductService) ReleaseStock(ctx context.Context, stockUpdates map[uint]int) error {
	// Add stocks back in a transaction
	err := s.productRepo.AddStocksInTransaction(stockUpdates)
	if err != nil {
		log.Printf("Failed to release stock: %v", err)
		return err
	}

	return nil
}