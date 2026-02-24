package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/unifiedpost/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// UnifiedPostRepository provides data access for the "unified_posts" collection.
type UnifiedPostRepository struct {
	*mongoRepo.MongoRepository[model.UnifiedPost]
}

// NewUnifiedPostRepository constructs a UnifiedPostRepository backed by the "unified_posts" collection.
func NewUnifiedPostRepository(db *mongo.Database) *UnifiedPostRepository {
	return &UnifiedPostRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.UnifiedPost](db, "unified_posts"),
	}
}

// FindNear returns UnifiedPosts within radiusKm of (lat, lng) using a $geoNear aggregation.
// Requires a 2dsphere index on the "location" field of "unified_posts".
func (r *UnifiedPostRepository) FindNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.UnifiedPost, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$geoNear", Value: bson.D{
			{Key: "near", Value: bson.D{
				{Key: "type", Value: "Point"},
				{Key: "coordinates", Value: bson.A{lng, lat}},
			}},
			{Key: "distanceField", Value: "distance_m"},
			{Key: "maxDistance", Value: radiusKm * 1000},
			{Key: "spherical", Value: true},
			{Key: "key", Value: "location"},
		}}},
	}
	cursor, err := r.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("UnifiedPostRepository.FindNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.UnifiedPost
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("UnifiedPostRepository.FindNear decode: %w", err)
	}
	return results, nil
}
