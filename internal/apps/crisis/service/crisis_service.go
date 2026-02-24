package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/crisis/model"
	"crisisecho/internal/apps/crisis/repository"
)

// CrisisService defines the public contract for the crisis domain.
type CrisisService interface {
	CreateCrisis(ctx context.Context, crisis *model.Crisis) error
	GetAllCrises(ctx context.Context) ([]*model.Crisis, error)
	GetVerifiedCrises(ctx context.Context) ([]*model.Crisis, error)
	GetNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Crisis, error)
	VerifyCrisis(ctx context.Context, id string) error
}

type crisisService struct {
	repo *repository.CrisisRepository
}

// NewCrisisService constructs a CrisisService with the given repository.
func NewCrisisService(repo *repository.CrisisRepository) CrisisService {
	return &crisisService{repo: repo}
}

func (s *crisisService) CreateCrisis(ctx context.Context, crisis *model.Crisis) error {
	if crisis.Sources == nil {
		crisis.Sources = []string{}
	}
	if crisis.ImageURLs == nil {
		crisis.ImageURLs = []string{}
	}
	if err := s.repo.Create(ctx, crisis); err != nil {
		return fmt.Errorf("CrisisService.CreateCrisis: %w", err)
	}
	return nil
}

func (s *crisisService) GetAllCrises(ctx context.Context) ([]*model.Crisis, error) {
	crises, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("CrisisService.GetAllCrises: %w", err)
	}
	return crises, nil
}

func (s *crisisService) GetVerifiedCrises(ctx context.Context) ([]*model.Crisis, error) {
	crises, err := s.repo.FindConfirmed(ctx)
	if err != nil {
		return nil, fmt.Errorf("CrisisService.GetVerifiedCrises: %w", err)
	}
	return crises, nil
}

func (s *crisisService) GetNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Crisis, error) {
	crises, err := s.repo.FindNear(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("CrisisService.GetNearby: %w", err)
	}
	return crises, nil
}

func (s *crisisService) VerifyCrisis(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("CrisisService.VerifyCrisis: invalid id %q: %w", id, err)
	}
	if err := s.repo.MarkConfirmed(ctx, oid); err != nil {
		return fmt.Errorf("CrisisService.VerifyCrisis: %w", err)
	}
	return nil
}
