package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/responder/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// ── Responder Repository ──────────────────────────────────────────────────────

// ResponderRepository provides data access for the responders collection.
type ResponderRepository struct {
	*mongoRepo.MongoRepository[model.Responder]
}

// NewResponderRepository constructs a ResponderRepository backed by "responders".
func NewResponderRepository(db *mongo.Database) *ResponderRepository {
	return &ResponderRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.Responder](db, "responders"),
	}
}

// FindByUserID returns the responder profile for the given Firebase UID.
func (r *ResponderRepository) FindByUserID(ctx context.Context, userID string) (*model.Responder, error) {
	filter := bson.D{{Key: "user_id", Value: userID}}
	result, err := r.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("ResponderRepository.FindByUserID: %w", err)
	}
	return result, nil
}

// FindNear returns active responders within radiusKm of (lat, lng) using $geoNear.
func (r *ResponderRepository) FindNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Responder, error) {
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
		return nil, fmt.Errorf("ResponderRepository.FindNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.Responder
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("ResponderRepository.FindNear decode: %w", err)
	}
	return results, nil
}

// ── Response Repository ───────────────────────────────────────────────────────

// ResponseRepository provides data access for the responses collection.
type ResponseRepository struct {
	*mongoRepo.MongoRepository[model.Response]
}

// NewResponseRepository constructs a ResponseRepository backed by "responses".
func NewResponseRepository(db *mongo.Database) *ResponseRepository {
	return &ResponseRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.Response](db, "responses"),
	}
}

// FindByResponderID returns all responses made by the given responder.
func (r *ResponseRepository) FindByResponderID(ctx context.Context, responderID primitive.ObjectID) ([]*model.Response, error) {
	filter := bson.D{{Key: "responder_id", Value: responderID}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("ResponseRepository.FindByResponderID: %w", err)
	}
	return results, nil
}

// UpdateStatus sets the status field on the given response.
func (r *ResponseRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string) (*model.Response, error) {
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}}
	return r.FindOneAndUpdate(ctx, filter, update)
}

// UpdateRating sets the rating field on the given response.
func (r *ResponseRepository) UpdateRating(ctx context.Context, id primitive.ObjectID, rating int) (*model.Response, error) {
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "rating", Value: rating}}}}
	return r.FindOneAndUpdate(ctx, filter, update)
}
