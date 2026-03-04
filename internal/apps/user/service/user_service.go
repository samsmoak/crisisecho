package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"crisisecho/internal/apps/user/model"
	"crisisecho/internal/apps/user/repository"
)

// UserService defines the public contract for the user domain.
type UserService interface {
	CreateUser(ctx context.Context, uid, email, name string) (*model.User, error)
	GetUser(ctx context.Context, uid string) (*model.User, error)
	GetOrCreateUser(ctx context.Context, uid, email, name, picture string) (*model.User, error)
	UpdateUser(ctx context.Context, uid, name string) (*model.User, error)
}

type userService struct {
	repo *repository.UserRepository
}

// NewUserService constructs a UserService with the given repository.
func NewUserService(repo *repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, uid, email, name string) (*model.User, error) {
	now := time.Now().UTC()
	user := &model.User{
		FirebaseUID: uid,
		Email:       email,
		Name:        name,
		Role:        "free",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("UserService.CreateUser: %w", err)
	}
	return user, nil
}

func (s *userService) GetUser(ctx context.Context, uid string) (*model.User, error) {
	user, err := s.repo.FindByFirebaseUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("UserService.GetUser: %w", err)
	}
	return user, nil
}

func (s *userService) GetOrCreateUser(ctx context.Context, uid, email, name, picture string) (*model.User, error) {
	now := time.Now().UTC()
	user := &model.User{
		FirebaseUID: uid,
		Email:       email,
		Name:        name,
		Picture:     picture,
		Role:        "free",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	result, err := s.repo.UpsertByFirebaseUID(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("UserService.GetOrCreateUser: %w", err)
	}
	return result, nil
}

func (s *userService) UpdateUser(ctx context.Context, uid, name string) (*model.User, error) {
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: name},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}
	result, err := s.repo.UpdateProfile(ctx, uid, update)
	if err != nil {
		return nil, fmt.Errorf("UserService.UpdateUser: %w", err)
	}
	return result, nil
}
