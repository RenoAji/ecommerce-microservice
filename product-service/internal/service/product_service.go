package service

import (
	"product-service/internal/domain"
	"product-service/internal/repository"
)

type ProductService struct {
	productRepo repository.ProductRepository
}

func NewProductService(pr repository.ProductRepository) *ProductService {
	return &ProductService{productRepo: pr}
}

func (s *ProductService) CreateProduct(product *domain.CreateProductRequest) error {
	return s.productRepo.SaveProduct(product)
}

func (s *ProductService) CreateCategory(category *domain.Category) error {
	return s.productRepo.CreateCategory(category)
}

type PaginatedProducts struct {
	Products    []domain.ProductResponse `json:"products"`
	Total       int64                    `json:"total"`
	Page        int                      `json:"page"`
	Limit       int                      `json:"limit"`
	TotalPages  int                      `json:"total_pages"`
}

func (s *ProductService) GetProducts(search, categoryID, minPrice, maxPrice, order, sortBy string, page, limit int) (*PaginatedProducts, error) {
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

	return &PaginatedProducts{
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
