package service

import (
	"errors"
	"testing"
	"user-service/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	savedUser         *domain.User
	saveErr           error
	findByIDUser      *domain.User
	findByIDErr       error
	updatePasswordID  uint
	updatedPassword   string
	updatePasswordErr error
}

func (m *mockUserRepository) Save(user *domain.User) error {
	m.savedUser = user
	return m.saveErr
}

func (m *mockUserRepository) FindByEmail(email string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) FindByID(userID uint) (*domain.User, error) {
	return m.findByIDUser, m.findByIDErr
}

func (m *mockUserRepository) UpdatePassword(userID uint, newPassword string) error {
	m.updatePasswordID = userID
	m.updatedPassword = newPassword
	return m.updatePasswordErr
}

func TestRegisterUserHashesPasswordBeforeSaving(t *testing.T) {
	repo := &mockUserRepository{}
	svc := NewUserService(repo)

	user := &domain.User{Email: "john@example.com", Password: "plain-password"}
	if err := svc.RegisterUser(user); err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	if repo.savedUser == nil {
		t.Fatal("expected user to be saved")
	}
	if repo.savedUser.Password == "plain-password" {
		t.Fatal("expected password to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.savedUser.Password), []byte("plain-password")); err != nil {
		t.Fatalf("saved password is not a valid bcrypt hash: %v", err)
	}
}

func TestChangePasswordRejectsInvalidOldPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-old"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	repo := &mockUserRepository{findByIDUser: &domain.User{ID: 10, Password: string(hash)}}
	svc := NewUserService(repo)

	err = svc.ChangePassword(10, "wrong-old", "new-password")
	if err == nil {
		t.Fatal("expected error for wrong old password")
	}
	if repo.updatedPassword != "" {
		t.Fatal("did not expect password update when old password is wrong")
	}
}

func TestChangePasswordUpdatesPasswordWhenOldPasswordMatches(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-old"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	repo := &mockUserRepository{findByIDUser: &domain.User{ID: 33, Password: string(hash)}}
	svc := NewUserService(repo)

	err = svc.ChangePassword(33, "correct-old", "new-password")
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	if repo.updatePasswordID != 33 {
		t.Fatalf("expected update password call for user 33, got %d", repo.updatePasswordID)
	}
	if repo.updatedPassword == "new-password" || repo.updatedPassword == "" {
		t.Fatalf("expected hashed new password, got %q", repo.updatedPassword)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.updatedPassword), []byte("new-password")); err != nil {
		t.Fatalf("updated password is not valid bcrypt hash: %v", err)
	}
}
