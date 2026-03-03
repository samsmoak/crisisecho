package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/responder/model"
	"crisisecho/internal/apps/responder/repository"
	"crisisecho/internal/geo"
)

// ResponderService defines the public contract for the responder domain.
type ResponderService interface {
	RegisterResponder(ctx context.Context, responder *model.Responder) error
	GetResponderByUser(ctx context.Context, userID string) (*model.Responder, error)
	UpdateResponder(ctx context.Context, id string, responder *model.Responder) error
	FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Responder, error)
	CreateResponse(ctx context.Context, response *model.Response) error
	UpdateResponseStatus(ctx context.Context, id, status string) (*model.Response, error)
	RateResponse(ctx context.Context, id string, rating int) (*model.Response, error)
}

type responderService struct {
	responderRepo *repository.ResponderRepository
	responseRepo  *repository.ResponseRepository
}

// NewResponderService constructs a ResponderService with the given repositories.
func NewResponderService(
	responderRepo *repository.ResponderRepository,
	responseRepo *repository.ResponseRepository,
) ResponderService {
	return &responderService{
		responderRepo: responderRepo,
		responseRepo:  responseRepo,
	}
}

func (s *responderService) RegisterResponder(ctx context.Context, responder *model.Responder) error {
	if responder.Capabilities == nil {
		responder.Capabilities = []string{}
	}
	responder.Location = geo.NewPoint(responder.Lat, responder.Lng)
	now := time.Now().UTC()
	responder.CreatedAt = now
	responder.UpdatedAt = now
	responder.Active = true
	if err := s.responderRepo.Create(ctx, responder); err != nil {
		return fmt.Errorf("ResponderService.RegisterResponder: %w", err)
	}
	return nil
}

func (s *responderService) GetResponderByUser(ctx context.Context, userID string) (*model.Responder, error) {
	responder, err := s.responderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ResponderService.GetResponderByUser: %w", err)
	}
	return responder, nil
}

func (s *responderService) UpdateResponder(ctx context.Context, id string, responder *model.Responder) error {
	responder.UpdatedAt = time.Now().UTC()
	responder.Location = geo.NewPoint(responder.Lat, responder.Lng)
	if err := s.responderRepo.Update(ctx, id, responder); err != nil {
		return fmt.Errorf("ResponderService.UpdateResponder: %w", err)
	}
	return nil
}

func (s *responderService) FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Responder, error) {
	responders, err := s.responderRepo.FindNear(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("ResponderService.FindNearby: %w", err)
	}
	return responders, nil
}

func (s *responderService) CreateResponse(ctx context.Context, response *model.Response) error {
	now := time.Now().UTC()
	response.StartedAt = &now
	if response.Status == "" {
		response.Status = "en_route"
	}
	if err := s.responseRepo.Create(ctx, response); err != nil {
		return fmt.Errorf("ResponderService.CreateResponse: %w", err)
	}
	return nil
}

func (s *responderService) UpdateResponseStatus(ctx context.Context, id, status string) (*model.Response, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("ResponderService.UpdateResponseStatus: invalid id %q: %w", id, err)
	}
	result, err := s.responseRepo.UpdateStatus(ctx, oid, status)
	if err != nil {
		return nil, fmt.Errorf("ResponderService.UpdateResponseStatus: %w", err)
	}
	return result, nil
}

func (s *responderService) RateResponse(ctx context.Context, id string, rating int) (*model.Response, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("ResponderService.RateResponse: invalid id %q: %w", id, err)
	}
	result, err := s.responseRepo.UpdateRating(ctx, oid, rating)
	if err != nil {
		return nil, fmt.Errorf("ResponderService.RateResponse: %w", err)
	}
	return result, nil
}
