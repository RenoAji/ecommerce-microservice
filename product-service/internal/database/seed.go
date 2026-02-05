package database

import (
	"log"
	"product-service/internal/domain"

	"gorm.io/gorm"
)

func SeedData(db *gorm.DB) {
	// Seed Categories
	categories := []domain.Category{
		{Name: "Electronics"},
		{Name: "Clothing"},
		{Name: "Books"},
		{Name: "Home & Kitchen"},
	}

	for i := range categories {
		var existing domain.Category
		if err := db.Where("name = ?", categories[i].Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&categories[i]).Error; err != nil {
				log.Printf("Failed to seed category %s: %v", categories[i].Name, err)
			} else {
				log.Printf("Category '%s' seeded successfully", categories[i].Name)
			}
		} else {
			categories[i] = existing // Use existing category
			log.Printf("Category '%s' already exists", categories[i].Name)
		}
	}

	// Seed Products
	products := []domain.Product{
		{
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse with USB receiver",
			Price:       2500000, // 25,000.00 in cents
			Stock:       50,
			Categories:  []domain.Category{categories[0]}, // Electronics
		},
		{
			Name:        "Mechanical Keyboard",
			Description: "RGB mechanical gaming keyboard",
			Price:       7500000, // 75,000.00
			Stock:       30,
			Categories:  []domain.Category{categories[0]},
		},
		{
			Name:        "Cotton T-Shirt",
			Description: "Comfortable 100% cotton t-shirt",
			Price:       1500000, // 15,000.00
			Stock:       100,
			Categories:  []domain.Category{categories[1]}, // Clothing
		},
		{
			Name:        "Denim Jeans",
			Description: "Classic blue denim jeans",
			Price:       4500000, // 45,000.00
			Stock:       75,
			Categories:  []domain.Category{categories[1]},
		},
		{
			Name:        "Programming Book",
			Description: "Learn Go programming from scratch",
			Price:       3500000, // 35,000.00
			Stock:       40,
			Categories:  []domain.Category{categories[2]}, // Books
		},
		{
			Name:        "Cooking Guide",
			Description: "Master chef cooking techniques",
			Price:       2800000, // 28,000.00
			Stock:       60,
			Categories:  []domain.Category{categories[2], categories[3]}, // Books & Home
		},
		{
			Name:        "Coffee Maker",
			Description: "Automatic drip coffee maker",
			Price:       8500000, // 85,000.00
			Stock:       20,
			Categories:  []domain.Category{categories[3]}, // Home & Kitchen
		},
		{
			Name:        "Blender",
			Description: "High-speed blender for smoothies",
			Price:       5500000, // 55,000.00
			Stock:       35,
			Categories:  []domain.Category{categories[3]},
		},
		{
			Name:        "USB-C Cable",
			Description: "Fast charging USB-C cable 2m",
			Price:       1200000, // 12,000.00
			Stock:       200,
			Categories:  []domain.Category{categories[0]},
		},
		{
			Name:        "Laptop Stand",
			Description: "Adjustable aluminum laptop stand",
			Price:       3200000, // 32,000.00
			Stock:       45,
			Categories:  []domain.Category{categories[0]},
		},
	}

	for i := range products {
		var existing domain.Product
		if err := db.Where("name = ?", products[i].Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&products[i]).Error; err != nil {
				log.Printf("Failed to seed product %s: %v", products[i].Name, err)
			} else {
				log.Printf("Product '%s' seeded successfully", products[i].Name)
			}
		} else {
			log.Printf("Product '%s' already exists", products[i].Name)
		}
	}

	log.Println("Database seeding completed!")
}

