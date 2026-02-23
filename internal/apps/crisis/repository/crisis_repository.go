package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/crisis/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// CrisisRepository provides data access for the crises collection.
type CrisisRepository struct {
	*mongoRepo.MongoRepository[model.Crisis]
}

// NewCrisisRepository constructs a CrisisRepository backed by the "crises" collection.
func NewCrisisRepository(db *mongo.Database) *CrisisRepository {
	return &CrisisRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.Crisis](db, "crises"),
	}
}

// FindConfirmed returns all crises where confirmed=true.
func (r *CrisisRepository) FindConfirmed(ctx context.Context) ([]*model.Crisis, error) {
	filter := bson.D{{Key: "confirmed", Value: true}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("CrisisRepository.FindConfirmed: %w", err)
	}
	return results, nil
}

// FindNear returns crises within radiusKm of (lat, lng) using a $geoNear aggregation.
// Requires a 2dsphere index on the "location" field.
func (r *CrisisRepository) FindNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Crisis, error) {
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
		return nil, fmt.Errorf("CrisisRepository.FindNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.Crisis
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("CrisisRepository.FindNear decode: %w", err)
	}
	return results, nil
}

// MarkConfirmed sets confirmed=true and updates last_updated on the given crisis.
func (r *CrisisRepository) MarkConfirmed(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "confirmed",    Value: true},
		{Key: "last_updated", Value: time.Now().UTC()},
	}}}
	res, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("CrisisRepository.MarkConfirmed: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("CrisisRepository.MarkConfirmed: crisis not found")
	}
	return nil
}
