package repository

import (
	"errors"
	"product-service/internal/domain"

	"gorm.io/gorm"
)

type ProductRepository interface {
	SaveProduct(product *domain.CreateProductRequest) error
	CreateCategory(category *domain.Category) error
	AddStock(productID uint, add int) error
	Delete(productID uint) error
	GetByID(productID uint) (*domain.Product, error)
	ListAll(category, min, max, search, order, sortBy string, page, limit int) ([]domain.Product, int64, error)
	AssignCategory(productID uint, categoryID []uint) error
	RemoveCategory(productID uint, categoryID uint) error
	ListCategories(productID uint) ([]domain.Category, error)
	UpdateProduct(id uint, req *domain.UpdateProductRequest) (*domain.Product, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CREATE

func (r *PostgresRepository) SaveProduct(req *domain.CreateProductRequest) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        
        if len(req.CategoryIDs) > 0 {
            var count int64
            err := tx.Model(&domain.Category{}).Where("id IN ?", req.CategoryIDs).Count(&count).Error
            if err != nil {
                return err
            }
            if int(count) != len(req.CategoryIDs) {
                return errors.New("one or more category IDs do not exist")
            }
        }

        categories := make([]domain.Category, len(req.CategoryIDs))
        for i, id := range req.CategoryIDs {
            categories[i] = domain.Category{ID: id}
        }

        product := domain.Product{
            Name:        req.Name,
            Description: req.Description,
            Price:       req.Price,
            Stock:       req.Stock,
            Categories:  categories,
        }

        if err := tx.Omit("Categories.*").Create(&product).Error; err != nil {
            return err
        }

        return nil
    })
}

func (r *PostgresRepository) CreateCategory(category *domain.Category) error {
	return r.db.Create(category).Error
}

// READ

// FilterByCategory filters products by category ID
func FilterByCategory(categoryID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if categoryID == "" {
			return db
		}
		// Join with the many-to-many table to filter
		return db.Joins("JOIN product_categories ON product_categories.product_id = products.id").
			Where("product_categories.category_id = ?", categoryID)
	}
}

// FilterByPriceRange filters products between min and max price
func FilterByPriceRange(min, max string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if min != "" {
			db = db.Where("price >= ?", min)
		}
		if max != "" {
			db = db.Where("price <= ?", max)
		}
		return db
	}
}

// SearchByName performs a case-insensitive search
func SearchByName(query string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if query == "" {
			return db
		}
		// ILIKE is specific to PostgreSQL for case-insensitive search
		return db.Where("products.name ILIKE ?", "%"+query+"%")
	}
}

func OrderBy(sortBy, order string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if order == "asc" {
			return db.Order(sortBy + " ASC")
		} else if order == "desc" {
			return db.Order(sortBy + " DESC")
		}
		return db
	}
}

func (r *PostgresRepository) GetByID(productID uint) (*domain.Product, error) {
	var product domain.Product
	result := r.db.First(&product, productID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &product, nil
}

func (r *PostgresRepository) ListAll(search, category, min, max, order, sortBy string, page, limit int) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64

	// Build base query
	query := r.db.Model(&domain.Product{}).
		Scopes(
			FilterByCategory(category),
			FilterByPriceRange(min, max),
			SearchByName(search),
		)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Apply pagination, ordering, and load data
	err := query.Preload("Categories").
		Scopes(OrderBy(sortBy, order)).
		Offset(offset).
		Limit(limit).
		Find(&products).Error

	return products, total, err
}

func (r *PostgresRepository) ListCategories(productID uint) ([]domain.Category, error) {
	var categories []domain.Category
	result := r.db.Joins("JOIN product_categories ON categories.id = product_categories.category_id").
		Where("product_categories.product_id = ?", productID).
		Find(&categories)
	if result.Error != nil {
		return nil, result.Error
	}
	return categories, nil
}

// UPDATE

func (r *PostgresRepository) UpdateProduct(id uint, req *domain.UpdateProductRequest) (*domain.Product, error) {
    var product domain.Product

    err := r.db.Transaction(func(tx *gorm.DB) error {
        // 1. Find existing product
        if err := tx.First(&product, id).Error; err != nil {
            return err
        }

        // 2. Update specific fields (Map logic)
        updates := make(map[string]interface{})
        if req.Name != nil { updates["name"] = *req.Name }
        if req.Description != nil { updates["description"] = *req.Description }
        if req.Price != nil { updates["price"] = *req.Price }
        if req.Stock != nil { updates["stock"] = *req.Stock }

        if len(updates) > 0 {
            if err := tx.Model(&product).Updates(updates).Error; err != nil {
                return err
            }
        }

        // 3. Handle Category Updates
        if req.CategoryIDs != nil {
            newCategories := make([]domain.Category, len(req.CategoryIDs))
            for i, catID := range req.CategoryIDs {
                newCategories[i] = domain.Category{ID: catID}
            }
            if err := tx.Model(&product).Association("Categories").Replace(newCategories); err != nil {
                return err
            }
        }

        // 4. IMPORTANT: Re-fetch the product with Categories to get the "Final" version
        return tx.Preload("Categories").First(&product, id).Error
    })

    return &product, err
}

func (r *PostgresRepository) AddStock(productID uint, add int) error {
	result := r.db.Model(&domain.Product{}).
		Where("id = ?", productID).
		Where("stock + ? >= 0", add).
		UpdateColumn("stock", gorm.Expr("stock + ?", add))

	if result.Error != nil {
		return result.Error
	}

	// If no rows were changed, it means the product ID is wrong OR the result would have been negative.
	if result.RowsAffected == 0 {
		return errors.New("product not found or resulting stock would be negative")
	}

	return nil
}

func (r *PostgresRepository) AssignCategory(productID uint, categoryIDs []uint) error {
	categories := make([]domain.Category, len(categoryIDs))
	for i, id := range categoryIDs {
		categories[i] = domain.Category{ID: id}
	}

	return r.db.Model(&domain.Product{ID: productID}).
		Omit("Categories.*").
		Association("Categories").
		Append(&categories)
}

func (r *PostgresRepository) RemoveCategory(productID uint, categoryID uint) error {
	return r.db.Model(&domain.Product{ID: productID}).
		Association("Categories").
		Delete(&domain.Category{ID: categoryID})
}

// DELETE

func (r *PostgresRepository) Delete(productID uint) error {
	// return r.db.Delete(&domain.Product{}, productID).Error
	result := r.db.Delete(&domain.Product{}, productID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
