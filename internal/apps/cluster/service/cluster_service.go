package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/cluster/model"
	"crisisecho/internal/apps/cluster/repository"
)

// ClusterService defines the public contract for the cluster domain.
type ClusterService interface {
	CreateCluster(ctx context.Context, cluster *model.Cluster) error
	GetHotspots(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Cluster, error)
	GetClusterDetail(ctx context.Context, id string) (*model.Cluster, error)
	UpdateClusterStatus(ctx context.Context, id primitive.ObjectID, status string) error
}

type clusterService struct {
	repo *repository.ClusterRepository
}

// NewClusterService constructs a ClusterService with the given repository.
func NewClusterService(repo *repository.ClusterRepository) ClusterService {
	return &clusterService{repo: repo}
}

func (s *clusterService) CreateCluster(ctx context.Context, cluster *model.Cluster) error {
	if cluster.PostIDs == nil {
		cluster.PostIDs = []primitive.ObjectID{}
	}
	if cluster.Sources == nil {
		cluster.Sources = []string{}
	}
	if err := s.repo.Create(ctx, cluster); err != nil {
		return fmt.Errorf("ClusterService.CreateCluster: %w", err)
	}
	return nil
}

func (s *clusterService) GetHotspots(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Cluster, error) {
	clusters, err := s.repo.FindActiveNear(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("ClusterService.GetHotspots: %w", err)
	}
	return clusters, nil
}

func (s *clusterService) GetClusterDetail(ctx context.Context, id string) (*model.Cluster, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ClusterService.GetClusterDetail: %w", err)
	}
	return cluster, nil
}

func (s *clusterService) UpdateClusterStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("ClusterService.UpdateClusterStatus: %w", err)
	}
	return nil
}
