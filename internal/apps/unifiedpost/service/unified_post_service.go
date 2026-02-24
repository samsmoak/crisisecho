package service

import (
	"context"
	"fmt"

	"crisisecho/internal/apps/unifiedpost/model"
	"crisisecho/internal/apps/unifiedpost/repository"
)

// UnifiedPostService defines the public contract for the unified post domain.
type UnifiedPostService interface {
	CreateUnifiedPost(ctx context.Context, post *model.UnifiedPost) error
	GetUnifiedPost(ctx context.Context, id string) (*model.UnifiedPost, error)
	GetNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.UnifiedPost, error)
}

type unifiedPostService struct {
	repo *repository.UnifiedPostRepository
}

// NewUnifiedPostService constructs a UnifiedPostService.
func NewUnifiedPostService(repo *repository.UnifiedPostRepository) UnifiedPostService {
	return &unifiedPostService{repo: repo}
}

func (s *unifiedPostService) CreateUnifiedPost(ctx context.Context, post *model.UnifiedPost) error {
	if post.Sources == nil {
		post.Sources = []string{}
	}
	if post.PostIDs == nil {
		post.PostIDs = []string{}
	}
	if err := s.repo.Create(ctx, post); err != nil {
		return fmt.Errorf("UnifiedPostService.CreateUnifiedPost: %w", err)
	}
	return nil
}

func (s *unifiedPostService) GetUnifiedPost(ctx context.Context, id string) (*model.UnifiedPost, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("UnifiedPostService.GetUnifiedPost: %w", err)
	}
	return post, nil
}

func (s *unifiedPostService) GetNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.UnifiedPost, error) {
	posts, err := s.repo.FindNear(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("UnifiedPostService.GetNearby: %w", err)
	}
	return posts, nil
}
