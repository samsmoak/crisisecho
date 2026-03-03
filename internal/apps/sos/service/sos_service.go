package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/sos/model"
	"crisisecho/internal/apps/sos/repository"
	"crisisecho/internal/geo"
)

const sosLiveChannel = "alerts:live"

// SOSService defines the public contract for the SOS domain.
type SOSService interface {
	CreateProfile(ctx context.Context, profile *model.SOSProfile) error
	GetProfilesByUser(ctx context.Context, userID string) ([]*model.SOSProfile, error)
	UpdateProfile(ctx context.Context, id string, profile *model.SOSProfile) error
	DeleteProfile(ctx context.Context, id string) error
	TriggerSOS(ctx context.Context, profileID, userID string, lat, lng float64) (*model.SOSAlert, error)
	ResolveSOSAlert(ctx context.Context, id string) error
	GetActiveAlerts(ctx context.Context, userID string) ([]*model.SOSAlert, error)
}

type sosService struct {
	profileRepo *repository.SOSProfileRepository
	alertRepo   *repository.SOSAlertRepository
	redisClient *redis.Client
}

// NewSOSService constructs an SOSService with the given repositories and Redis client.
func NewSOSService(
	profileRepo *repository.SOSProfileRepository,
	alertRepo *repository.SOSAlertRepository,
	redisClient *redis.Client,
) SOSService {
	return &sosService{
		profileRepo: profileRepo,
		alertRepo:   alertRepo,
		redisClient: redisClient,
	}
}

func (s *sosService) CreateProfile(ctx context.Context, profile *model.SOSProfile) error {
	if profile.EmergencyContacts == nil {
		profile.EmergencyContacts = []model.EmergencyContact{}
	}
	if profile.Severity == 0 {
		profile.Severity = 5
	}
	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	profile.Active = true
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return fmt.Errorf("SOSService.CreateProfile: %w", err)
	}
	return nil
}

func (s *sosService) GetProfilesByUser(ctx context.Context, userID string) ([]*model.SOSProfile, error) {
	profiles, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SOSService.GetProfilesByUser: %w", err)
	}
	return profiles, nil
}

func (s *sosService) UpdateProfile(ctx context.Context, id string, profile *model.SOSProfile) error {
	profile.UpdatedAt = time.Now().UTC()
	if err := s.profileRepo.Update(ctx, id, profile); err != nil {
		return fmt.Errorf("SOSService.UpdateProfile: %w", err)
	}
	return nil
}

func (s *sosService) DeleteProfile(ctx context.Context, id string) error {
	if err := s.profileRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("SOSService.DeleteProfile: %w", err)
	}
	return nil
}

// TriggerSOS creates an SOSAlert from the given profile and publishes it to Redis.
func (s *sosService) TriggerSOS(ctx context.Context, profileID, userID string, lat, lng float64) (*model.SOSAlert, error) {
	oid, err := primitive.ObjectIDFromHex(profileID)
	if err != nil {
		return nil, fmt.Errorf("SOSService.TriggerSOS: invalid profileID %q: %w", profileID, err)
	}
	profile, err := s.profileRepo.GetByID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("SOSService.TriggerSOS: %w", err)
	}

	alert := &model.SOSAlert{
		UserID:          userID,
		SOSProfileID:    oid,
		Label:           profile.Label,
		EventType:       profile.EventType,
		Location:        geo.NewPoint(lat, lng),
		Lat:             lat,
		Lng:             lng,
		MessageTemplate: profile.MessageTemplate,
		Severity:        profile.Severity,
		Status:          "active",
		TriggeredAt:     time.Now().UTC(),
	}
	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("SOSService.TriggerSOS: %w", err)
	}

	// Publish to Redis — non-critical, log on error but don't fail the DB write.
	payload := map[string]any{
		"id":           alert.ID.Hex(),
		"alert_kind":   "sos",
		"label":        alert.Label,
		"event_type":   alert.EventType,
		"severity":     alert.Severity,
		"lat":          lat,
		"lng":          lng,
		"triggered_at": alert.TriggeredAt,
	}
	alertJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("SOSService.TriggerSOS: marshal error (non-critical): %v", err)
		return alert, nil
	}
	if err := s.redisClient.Publish(ctx, sosLiveChannel, alertJSON).Err(); err != nil {
		log.Printf("SOSService.TriggerSOS: redis publish error (non-critical): %v", err)
	}
	return alert, nil
}

func (s *sosService) ResolveSOSAlert(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("SOSService.ResolveSOSAlert: invalid id %q: %w", id, err)
	}
	if err := s.alertRepo.MarkResolved(ctx, oid); err != nil {
		return fmt.Errorf("SOSService.ResolveSOSAlert: %w", err)
	}
	return nil
}

func (s *sosService) GetActiveAlerts(ctx context.Context, userID string) ([]*model.SOSAlert, error) {
	alerts, err := s.alertRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SOSService.GetActiveAlerts: %w", err)
	}
	return alerts, nil
}
