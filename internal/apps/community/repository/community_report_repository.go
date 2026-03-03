package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/community/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// CommunityReportRepository provides data access for the community_reports collection.
type CommunityReportRepository struct {
	*mongoRepo.MongoRepository[model.CommunityReport]
}

// NewCommunityReportRepository constructs a CommunityReportRepository.
func NewCommunityReportRepository(db *mongo.Database) *CommunityReportRepository {
	return &CommunityReportRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.CommunityReport](db, "community_reports"),
	}
}

// FindNear returns community reports within radiusKm of (lat, lng) using $geoNear.
func (r *CommunityReportRepository) FindNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.CommunityReport, error) {
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
		return nil, fmt.Errorf("CommunityReportRepository.FindNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.CommunityReport
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("CommunityReportRepository.FindNear decode: %w", err)
	}
	return results, nil
}

// FindByUserID returns all reports submitted by the given user.
func (r *CommunityReportRepository) FindByUserID(ctx context.Context, userID string) ([]*model.CommunityReport, error) {
	filter := bson.D{{Key: "user_id", Value: userID}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("CommunityReportRepository.FindByUserID: %w", err)
	}
	return results, nil
}
