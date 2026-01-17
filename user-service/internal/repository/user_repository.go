package repository

import (
	"user-service/internal/domain"

	"gorm.io/gorm"
)

type UserRepository interface {
	Save(user *domain.User) error
	FindByEmail(email string) (*domain.User, error)
	FindByID(userID uint) (*domain.User, error)
	UpdatePassword(userID uint, newPassword string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

type MockRepository struct {
	users []domain.User
}

// POSTGRES REPOSITORY IMPLEMENTATION

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *PostgresRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *PostgresRepository) FindByID(userID uint) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *PostgresRepository) UpdatePassword(userID uint, newPassword string) error {
	{
		return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("password", newPassword).Error
	}
}

// MOCK REPOSITORY IMPLEMENTATION
// func NewMockRepository() UserRepository {
// 	return &MockRepository{users: []domain.User{}}
// }

// func (r *MockRepository) Save(user *domain.User) error {
// 	// Simulate saving to a list
// 	r.users = append(r.users, *user)
// 	return nil
// }
