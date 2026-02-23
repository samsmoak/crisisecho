package service

import (
	"context"
	"fmt"

	alertModel   "crisisecho/internal/apps/alert/model"
	clusterModel "crisisecho/internal/apps/cluster/model"
	"crisisecho/internal/apps/notify/model"
	"crisisecho/internal/apps/notify/repository"
)

// Threshold constants for alert publication decisions.
const (
	ThresholdSeverity           = 3
	ThresholdContributorCount   = 3
	ThresholdSourceCount        = 2
	ThresholdLocationConfidence = 0.5
)

// NotifyService defines the public contract for the notify domain.
type NotifyService interface {
	Subscribe(ctx context.Context, sub *model.Subscription) error
	Unsubscribe(ctx context.Context, id string) error
	GetSubscribersForAlert(ctx context.Context, alert *alertModel.Alert) ([]*model.Subscription, error)
	CheckThreshold(cluster *clusterModel.Cluster) bool
}

type notifyService struct {
	repo *repository.SubscriptionRepository
}

// NewNotifyService constructs a NotifyService with the given repository.
func NewNotifyService(repo *repository.SubscriptionRepository) NotifyService {
	return &notifyService{repo: repo}
}

func (s *notifyService) Subscribe(ctx context.Context, sub *model.Subscription) error {
	if sub.CrisisTypes == nil {
		sub.CrisisTypes = []string{}
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return fmt.Errorf("NotifyService.Subscribe: %w", err)
	}
	return nil
}

func (s *notifyService) Unsubscribe(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("NotifyService.Unsubscribe: %w", err)
	}
	return nil
}

// GetSubscribersForAlert returns active subscriptions near the alert's centroid.
// Uses a 100 km search radius around the centroid as a coarse filter;
// per-subscription radius matching can be applied by the caller if needed.
func (s *notifyService) GetSubscribersForAlert(ctx context.Context, alert *alertModel.Alert) ([]*model.Subscription, error) {
	lat := alert.Centroid.Lat()
	lng := alert.Centroid.Lng()
	subs, err := s.repo.FindActiveNear(ctx, lat, lng, 100)
	if err != nil {
		return nil, fmt.Errorf("NotifyService.GetSubscribersForAlert: %w", err)
	}
	return subs, nil
}

// CheckThreshold returns true when the cluster meets all publication thresholds:
//   - Severity >= 3
//   - ContributorCount >= 3 (distinct user accounts)
//   - len(Sources) >= 2 (distinct platforms)
//   - LocationConfidence >= 0.5
func (s *notifyService) CheckThreshold(cluster *clusterModel.Cluster) bool {
	return cluster.Severity >= ThresholdSeverity &&
		cluster.ContributorCount >= ThresholdContributorCount &&
		len(cluster.Sources) >= ThresholdSourceCount &&
		cluster.LocationConfidence >= ThresholdLocationConfidence
}
