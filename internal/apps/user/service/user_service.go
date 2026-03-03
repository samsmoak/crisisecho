package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"crisisecho/internal/apps/user/model"
	"crisisecho/internal/apps/user/repository"
	"crisisecho/internal/geo"
)

// UserService defines the public contract for the user domain.
type UserService interface {
	CreateUser(ctx context.Context, uid, email, name string) (*model.User, error)
	GetUser(ctx context.Context, uid string) (*model.User, error)
	GetOrCreateUser(ctx context.Context, uid, email, name, picture string) (*model.User, error)
	UpdateUser(ctx context.Context, uid, name string, savedLocations []model.SavedLocation) (*model.User, error)
	GetSavedLocations(ctx context.Context, uid string) ([]model.SavedLocation, error)
	AddSavedLocation(ctx context.Context, uid string, loc model.SavedLocation) (*model.User, error)
	RemoveSavedLocation(ctx context.Context, uid, label string) (*model.User, error)
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
		FirebaseUID:    uid,
		Email:          email,
		Name:           name,
		Role:           "free",
		SavedLocations: []model.SavedLocation{},
		CreatedAt:      now,
		UpdatedAt:      now,
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
		FirebaseUID:    uid,
		Email:          email,
		Name:           name,
		Picture:        picture,
		Role:           "free",
		SavedLocations: []model.SavedLocation{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	result, err := s.repo.UpsertByFirebaseUID(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("UserService.GetOrCreateUser: %w", err)
	}
	return result, nil
}

func (s *userService) UpdateUser(ctx context.Context, uid, name string, savedLocations []model.SavedLocation) (*model.User, error) {
	if savedLocations == nil {
		savedLocations = []model.SavedLocation{}
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: name},
		{Key: "saved_locations", Value: savedLocations},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}
	result, err := s.repo.UpdateProfile(ctx, uid, update)
	if err != nil {
		return nil, fmt.Errorf("UserService.UpdateUser: %w", err)
	}
	return result, nil
}

func (s *userService) GetSavedLocations(ctx context.Context, uid string) ([]model.SavedLocation, error) {
	user, err := s.repo.FindByFirebaseUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("UserService.GetSavedLocations: %w", err)
	}
	if user.SavedLocations == nil {
		return []model.SavedLocation{}, nil
	}
	return user.SavedLocations, nil
}

func (s *userService) AddSavedLocation(ctx context.Context, uid string, loc model.SavedLocation) (*model.User, error) {
	loc.Location = geo.NewPoint(loc.Lat, loc.Lng)
	update := bson.D{{Key: "$push", Value: bson.D{
		{Key: "saved_locations", Value: loc},
	}}}
	result, err := s.repo.UpdateProfile(ctx, uid, update)
	if err != nil {
		return nil, fmt.Errorf("UserService.AddSavedLocation: %w", err)
	}
	return result, nil
}

func (s *userService) RemoveSavedLocation(ctx context.Context, uid, label string) (*model.User, error) {
	update := bson.D{{Key: "$pull", Value: bson.D{
		{Key: "saved_locations", Value: bson.D{{Key: "label", Value: label}}},
	}}}
	result, err := s.repo.UpdateProfile(ctx, uid, update)
	if err != nil {
		return nil, fmt.Errorf("UserService.RemoveSavedLocation: %w", err)
	}
	return result, nil
}
