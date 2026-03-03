package service

import (
	"context"
	"fmt"
	"time"

	"crisisecho/internal/apps/community/model"
	"crisisecho/internal/apps/community/repository"
	"crisisecho/internal/geo"
)

// CommunityService defines the public contract for the community reports domain.
type CommunityService interface {
	CreateReport(ctx context.Context, report *model.CommunityReport) error
	GetNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.CommunityReport, error)
	GetByID(ctx context.Context, id string) (*model.CommunityReport, error)
	GetAll(ctx context.Context) ([]*model.CommunityReport, error)
}

type communityService struct {
	repo *repository.CommunityReportRepository
}

// NewCommunityService constructs a CommunityService with the given repository.
func NewCommunityService(repo *repository.CommunityReportRepository) CommunityService {
	return &communityService{repo: repo}
}

func (s *communityService) CreateReport(ctx context.Context, report *model.CommunityReport) error {
	if report.MediaURLs == nil {
		report.MediaURLs = []string{}
	}
	report.Location = geo.NewPoint(report.Lat, report.Lng)
	report.CreatedAt = time.Now().UTC()
	if report.Status == "" {
		report.Status = "pending"
	}
	if err := s.repo.Create(ctx, report); err != nil {
		return fmt.Errorf("CommunityService.CreateReport: %w", err)
	}
	return nil
}

func (s *communityService) GetNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.CommunityReport, error) {
	reports, err := s.repo.FindNear(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("CommunityService.GetNearby: %w", err)
	}
	return reports, nil
}

func (s *communityService) GetByID(ctx context.Context, id string) (*model.CommunityReport, error) {
	report, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("CommunityService.GetByID: %w", err)
	}
	return report, nil
}

func (s *communityService) GetAll(ctx context.Context) ([]*model.CommunityReport, error) {
	reports, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("CommunityService.GetAll: %w", err)
	}
	return reports, nil
}
