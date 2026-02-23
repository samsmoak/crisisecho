package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/alert/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// AlertRepository provides data access for the alerts collection.
type AlertRepository struct {
	*mongoRepo.MongoRepository[model.Alert]
}

// NewAlertRepository constructs an AlertRepository backed by the "alerts" collection.
func NewAlertRepository(db *mongo.Database) *AlertRepository {
	return &AlertRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.Alert](db, "alerts"),
	}
}

// FindRecent returns alerts published within the last `hours` hours.
func (r *AlertRepository) FindRecent(ctx context.Context, hours int) ([]*model.Alert, error) {
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	filter := bson.D{{Key: "published_at", Value: bson.D{{Key: "$gte", Value: since}}}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("AlertRepository.FindRecent: %w", err)
	}
	return results, nil
}

// FindBySeverity returns all alerts with severity >= minSeverity.
func (r *AlertRepository) FindBySeverity(ctx context.Context, minSeverity int) ([]*model.Alert, error) {
	filter := bson.D{{Key: "severity", Value: bson.D{{Key: "$gte", Value: minSeverity}}}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("AlertRepository.FindBySeverity: %w", err)
	}
	return results, nil
}
