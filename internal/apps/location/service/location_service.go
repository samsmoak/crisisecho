package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/location/model"
	"crisisecho/internal/apps/location/repository"
	"crisisecho/internal/geo"
)

type LocationService interface {
	Create(ctx context.Context, userID primitive.ObjectID, label string, lat, lng, radiusKm float64) (*model.SavedLocation, error)
	GetByUser(ctx context.Context, userID primitive.ObjectID) ([]*model.SavedLocation, error)
	GetByID(ctx context.Context, id string) (*model.SavedLocation, error)
	Update(ctx context.Context, id string, userID primitive.ObjectID, label string, lat, lng, radiusKm float64) (*model.SavedLocation, error)
	Delete(ctx context.Context, id string, userID primitive.ObjectID) error
}

type locationService struct {
	repo *repository.LocationRepository
}

func NewLocationService(repo *repository.LocationRepository) LocationService {
	return &locationService{repo: repo}
}

func (s *locationService) Create(ctx context.Context, userID primitive.ObjectID, label string, lat, lng, radiusKm float64) (*model.SavedLocation, error) {
	now := time.Now().UTC()
	loc := &model.SavedLocation{
		UserID:    userID,
		Label:     label,
		Location:  geo.NewPoint(lat, lng),
		Lat:       lat,
		Lng:       lng,
		RadiusKm:  radiusKm,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, loc); err != nil {
		return nil, fmt.Errorf("LocationService.Create: %w", err)
	}
	return loc, nil
}

func (s *locationService) GetByUser(ctx context.Context, userID primitive.ObjectID) ([]*model.SavedLocation, error) {
	locs, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("LocationService.GetByUser: %w", err)
	}
	return locs, nil
}

func (s *locationService) GetByID(ctx context.Context, id string) (*model.SavedLocation, error) {
	loc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("LocationService.GetByID: %w", err)
	}
	return loc, nil
}

func (s *locationService) Update(ctx context.Context, id string, userID primitive.ObjectID, label string, lat, lng, radiusKm float64) (*model.SavedLocation, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("LocationService.Update: invalid id %q: %w", id, err)
	}

	now := time.Now().UTC()
	loc := &model.SavedLocation{
		ID:        oid,
		UserID:    userID,
		Label:     label,
		Location:  geo.NewPoint(lat, lng),
		Lat:       lat,
		Lng:       lng,
		RadiusKm:  radiusKm,
		UpdatedAt: now,
	}
	if err := s.repo.Update(ctx, id, loc); err != nil {
		return nil, fmt.Errorf("LocationService.Update: %w", err)
	}
	return loc, nil
}

func (s *locationService) Delete(ctx context.Context, id string, userID primitive.ObjectID) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("LocationService.Delete: invalid id %q: %w", id, err)
	}
	if err := s.repo.DeleteByIDAndUserID(ctx, oid, userID); err != nil {
		return fmt.Errorf("LocationService.Delete: %w", err)
	}
	return nil
}
