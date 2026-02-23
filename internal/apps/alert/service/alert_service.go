package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"crisisecho/internal/apps/alert/model"
	"crisisecho/internal/apps/alert/repository"
)

// AlertsLiveChannel is the Redis Pub/Sub channel for live alert broadcasts.
const AlertsLiveChannel = "alerts:live"

// AlertService defines the public contract for the alert domain.
type AlertService interface {
	PublishAlert(ctx context.Context, alert *model.Alert) error
	GetAllAlerts(ctx context.Context) ([]*model.Alert, error)
	GetRecentAlerts(ctx context.Context, hours int) ([]*model.Alert, error)
}

type alertService struct {
	repo        *repository.AlertRepository
	redisClient *redis.Client
}

// NewAlertService constructs an AlertService with the given repository and Redis client.
func NewAlertService(repo *repository.AlertRepository, redisClient *redis.Client) AlertService {
	return &alertService{
		repo:        repo,
		redisClient: redisClient,
	}
}

// PublishAlert saves the alert to MongoDB then broadcasts it to the Redis alerts:live channel.
// The Redis publish is non-critical: errors are logged but do not fail the DB write.
func (s *alertService) PublishAlert(ctx context.Context, alert *model.Alert) error {
	if alert.SourcePlatforms == nil {
		alert.SourcePlatforms = []string{}
	}
	if alert.NotifiedUsers == nil {
		alert.NotifiedUsers = []string{}
	}
	if err := s.repo.Create(ctx, alert); err != nil {
		return fmt.Errorf("AlertService.PublishAlert: %w", err)
	}

	alertJSON, err := json.Marshal(alert)
	if err != nil {
		log.Printf("AlertService.PublishAlert: marshal error (non-critical): %v", err)
		return nil
	}
	if err := s.redisClient.Publish(ctx, AlertsLiveChannel, alertJSON).Err(); err != nil {
		log.Printf("AlertService.PublishAlert: redis publish error (non-critical): %v", err)
	}
	return nil
}

func (s *alertService) GetAllAlerts(ctx context.Context) ([]*model.Alert, error) {
	alerts, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("AlertService.GetAllAlerts: %w", err)
	}
	return alerts, nil
}

func (s *alertService) GetRecentAlerts(ctx context.Context, hours int) ([]*model.Alert, error) {
	alerts, err := s.repo.FindRecent(ctx, hours)
	if err != nil {
		return nil, fmt.Errorf("AlertService.GetRecentAlerts: %w", err)
	}
	return alerts, nil
}
