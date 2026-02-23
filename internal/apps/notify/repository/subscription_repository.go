package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/notify/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// SubscriptionRepository provides data access for the subscriptions collection.
type SubscriptionRepository struct {
	*mongoRepo.MongoRepository[model.Subscription]
}

// NewSubscriptionRepository constructs a SubscriptionRepository backed by the "subscriptions" collection.
func NewSubscriptionRepository(db *mongo.Database) *SubscriptionRepository {
	return &SubscriptionRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.Subscription](db, "subscriptions"),
	}
}

// FindActiveNear returns active subscriptions whose location is within radiusKm of (lat, lng).
// Uses $geoNear with a pre-filter on active=true.
// Requires a 2dsphere index on the "location" field.
func (r *SubscriptionRepository) FindActiveNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Subscription, error) {
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
			{Key: "query", Value: bson.D{{Key: "active", Value: true}}},
		}}},
	}
	cursor, err := r.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepository.FindActiveNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.Subscription
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("SubscriptionRepository.FindActiveNear decode: %w", err)
	}
	return results, nil
}

// FindByCrisisType returns active subscriptions that include the given crisis_type.
// Uses $in to match against the crisis_types array field.
// A subscription with an empty crisis_types array is intended to receive all types
// and is NOT returned by this method — callers must handle that separately.
func (r *SubscriptionRepository) FindByCrisisType(ctx context.Context, crisisType string) ([]*model.Subscription, error) {
	filter := bson.D{
		{Key: "crisis_types", Value: bson.D{{Key: "$in", Value: bson.A{crisisType}}}},
		{Key: "active",       Value: true},
	}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepository.FindByCrisisType: %w", err)
	}
	return results, nil
}
