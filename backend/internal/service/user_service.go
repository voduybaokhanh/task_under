package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type UserService interface {
	GetOrCreateUser(ctx context.Context, deviceID string) (*domain.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetOrCreateUser(ctx context.Context, deviceID string) (*domain.User, error) {
	if deviceID == "" {
		return nil, errors.New("device_id is required")
	}
	return s.userRepo.GetOrCreateByDeviceID(ctx, deviceID)
}

func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
