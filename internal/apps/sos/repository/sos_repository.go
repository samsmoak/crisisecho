package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/sos/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// ── SOSProfile Repository ─────────────────────────────────────────────────────

// SOSProfileRepository provides data access for the sos_profiles collection.
type SOSProfileRepository struct {
	*mongoRepo.MongoRepository[model.SOSProfile]
}

// NewSOSProfileRepository constructs a SOSProfileRepository backed by "sos_profiles".
func NewSOSProfileRepository(db *mongo.Database) *SOSProfileRepository {
	return &SOSProfileRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.SOSProfile](db, "sos_profiles"),
	}
}

// FindByUserID returns all SOS profiles belonging to the given user.
func (r *SOSProfileRepository) FindByUserID(ctx context.Context, userID string) ([]*model.SOSProfile, error) {
	filter := bson.D{{Key: "user_id", Value: userID}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("SOSProfileRepository.FindByUserID: %w", err)
	}
	return results, nil
}

// ── SOSAlert Repository ───────────────────────────────────────────────────────

// SOSAlertRepository provides data access for the sos_alerts collection.
type SOSAlertRepository struct {
	*mongoRepo.MongoRepository[model.SOSAlert]
}

// NewSOSAlertRepository constructs a SOSAlertRepository backed by "sos_alerts".
func NewSOSAlertRepository(db *mongo.Database) *SOSAlertRepository {
	return &SOSAlertRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.SOSAlert](db, "sos_alerts"),
	}
}

// FindActiveByUserID returns SOS alerts with status="active" for the given user.
func (r *SOSAlertRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*model.SOSAlert, error) {
	filter := bson.D{
		{Key: "user_id", Value: userID},
		{Key: "status", Value: "active"},
	}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("SOSAlertRepository.FindActiveByUserID: %w", err)
	}
	return results, nil
}

// MarkResolved sets status="resolved" and resolved_at on the given SOS alert.
func (r *SOSAlertRepository) MarkResolved(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "status", Value: "resolved"},
		{Key: "resolved_at", Value: now},
	}}}
	res, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("SOSAlertRepository.MarkResolved: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("SOSAlertRepository.MarkResolved: alert not found")
	}
	return nil
}
