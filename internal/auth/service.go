package auth

import (
	"errors"
	"fmt"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/models"
)

type Service struct {
	*Auth
	db *database.DB
}

func NewService(a *Auth, db *database.DB) *Service {
	return &Service{Auth: a, db: db}
}

func (s *Service) Authenticate(username, password string) (*models.User, error) {
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.Active {
		return nil, errors.New("account is disabled")
	}

	if !s.VerifyPassword(user.PasswordHash, password) {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (s *Service) GetUserByID(id int64) (*models.User, error) {
	return s.db.GetUserByID(id)
}

func (s *Service) ValidateAPIKey(key string) (*models.User, error) {
	return s.db.GetUserByAPIKey(key)
}

func (s *Service) CreateUser(username, email, password string, role models.Role) (*models.User, error) {
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	apiKey, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		APIKey:       apiKey,
		Active:       true,
	}

	if err := s.db.CreateUser(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
