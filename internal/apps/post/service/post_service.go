package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/post/model"
	"crisisecho/internal/apps/post/repository"
	"crisisecho/internal/database"
)

// PostService defines the public contract for the post domain.
type PostService interface {
	CreateRawPost(ctx context.Context, source string, post *model.RawPost) error
	CreateUnifiedPost(ctx context.Context, post *model.UnifiedPost) error
	GetNearbyPosts(ctx context.Context, lat, lng, radiusKm float64) ([]*model.UnifiedPost, error)
	GetRecentPosts(ctx context.Context, minutes int) ([]*model.UnifiedPost, error)
	UpdateClusterID(ctx context.Context, postID string, clusterID primitive.ObjectID) error
}

type postService struct {
	db          *mongo.Database
	unifiedRepo *repository.UnifiedPostRepository
}

// NewPostService constructs a PostService.
// db is held for on-demand RawPostRepository creation (keyed by source at call time).
func NewPostService(db *mongo.Database, unifiedRepo *repository.UnifiedPostRepository) PostService {
	return &postService{
		db:          db,
		unifiedRepo: unifiedRepo,
	}
}

// rawRepoForSource creates a RawPostRepository for the given source on demand.
// The collection name is resolved via database.CollectionNameForSource to stay
// consistent with the mapping in database.go without duplicating the map.
func (s *postService) rawRepoForSource(source string) *repository.RawPostRepository {
	return repository.NewRawPostRepository(s.db, database.CollectionNameForSource(source))
}

func (s *postService) CreateRawPost(ctx context.Context, source string, post *model.RawPost) error {
	if post.ImageURLs == nil {
		post.ImageURLs = []string{}
	}
	repo := s.rawRepoForSource(source)
	if err := repo.Create(ctx, post); err != nil {
		return fmt.Errorf("PostService.CreateRawPost: %w", err)
	}
	return nil
}

func (s *postService) CreateUnifiedPost(ctx context.Context, post *model.UnifiedPost) error {
	if post.ImageURLs == nil {
		post.ImageURLs = []string{}
	}
	if post.ImageEmbeddingIDs == nil {
		post.ImageEmbeddingIDs = []string{}
	}
	if err := s.unifiedRepo.Create(ctx, post); err != nil {
		return fmt.Errorf("PostService.CreateUnifiedPost: %w", err)
	}
	return nil
}

func (s *postService) GetNearbyPosts(ctx context.Context, lat, lng, radiusKm float64) ([]*model.UnifiedPost, error) {
	posts, err := s.unifiedRepo.FindNear(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("PostService.GetNearbyPosts: %w", err)
	}
	return posts, nil
}

func (s *postService) GetRecentPosts(ctx context.Context, minutes int) ([]*model.UnifiedPost, error) {
	posts, err := s.unifiedRepo.FindRecentRelevant(ctx, minutes)
	if err != nil {
		return nil, fmt.Errorf("PostService.GetRecentPosts: %w", err)
	}
	return posts, nil
}

func (s *postService) UpdateClusterID(ctx context.Context, postID string, clusterID primitive.ObjectID) error {
	if err := s.unifiedRepo.UpdateClusterID(ctx, postID, clusterID); err != nil {
		return fmt.Errorf("PostService.UpdateClusterID: %w", err)
	}
	return nil
}
