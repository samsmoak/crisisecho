package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"crisisecho/internal/apps/analytics/repository"
)

// AnalyticsService defines the public contract for the analytics domain.
type AnalyticsService interface {
	GetTrend(ctx context.Context, eventType string, days int) ([]bson.M, error)
	GetHeatmap(ctx context.Context, lat, lng, radiusKm float64) ([]bson.M, error)
}

type analyticsService struct {
	repo *repository.AnalyticsRepository
}

// NewAnalyticsService constructs an AnalyticsService with the given repository.
func NewAnalyticsService(repo *repository.AnalyticsRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

func (s *analyticsService) GetTrend(ctx context.Context, eventType string, days int) ([]bson.M, error) {
	if days <= 0 {
		days = 30
	}
	results, err := s.repo.AggregateTrend(ctx, eventType, days)
	if err != nil {
		return nil, fmt.Errorf("AnalyticsService.GetTrend: %w", err)
	}
	return results, nil
}

func (s *analyticsService) GetHeatmap(ctx context.Context, lat, lng, radiusKm float64) ([]bson.M, error) {
	results, err := s.repo.AggregateHeatmap(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("AnalyticsService.GetHeatmap: %w", err)
	}
	return results, nil
}
