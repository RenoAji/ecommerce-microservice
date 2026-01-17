package database

import (
	"log"
	"user-service/internal/domain"
	"user-service/internal/service"

	"gorm.io/gorm"
)

func SeedAdmin(db *gorm.DB, svc *service.UserService) {
	adminEmail := "admin@domain.com"

	// Check if admin user already exists
	var existingUser domain.User
	result := db.Where("email = ?", adminEmail).First(&existingUser)
	if result.Error == nil {
		log.Println("Admin user already exists, skipping seeding.")
		return
	}

	adminUser := &domain.User{
		Email:    adminEmail,
		Password: "admin123", // In production, use a more secure password and consider environment variables
		Username: "admin",
		Role:     "admin",
	}

	if err := svc.RegisterUser(adminUser); err != nil {
		log.Println("Failed to seed admin user:", err)
		return
	}

	log.Println("Admin user seeded successfully.")
}
