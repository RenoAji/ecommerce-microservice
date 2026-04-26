package service

import (
	"context"
	"errors"
	"fmt"
	"libs/logger"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) RegisterUser(ctx context.Context, user *domain.User) error {
	l := logger.ForContext(ctx)
	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		l.Error("failed to hash password", zap.Error(err))
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now()

	err = s.repo.Save(user)
	if err != nil {
		l.Error("failed to save user", zap.Error(err))
		return fmt.Errorf("failed to save user: %w", err)
	}
	l.Info("User registered successfully", zap.String("email", user.Email))
	return nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	l := logger.ForContext(ctx)
	// Fetch user by ID (you might need to implement a FindByID method in the repository)
	user, err := s.repo.FindByID(userID)
	if err != nil {
		l.Error("failed to find user by id", zap.Error(err))
		return fmt.Errorf("failed to find user by id: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("incorrect old password") // Old password does not match
	}

	// Hash new password
	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		l.Error("failed to hash new password", zap.Error(err))
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in the repository
	err = s.repo.UpdatePassword(userID, string(hashedNewPassword))
	if err != nil {
		l.Error("failed to update password", zap.Error(err))
		return fmt.Errorf("failed to update password: %w", err)
	}
	l.Info("User password changed successfully", zap.Uint("userID", userID))
	return nil
}
