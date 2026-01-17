package service

import (
	"errors"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) RegisterUser(user *domain.User) error {
	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now()

	return s.repo.Save(user)
}

func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// Fetch user by ID (you might need to implement a FindByID method in the repository)
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("incorrect old password") // Old password does not match
	}

	// Hash new password
	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password in the repository
	return s.repo.UpdatePassword(userID, string(hashedNewPassword))
}
