package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/analytics/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// AnalyticsRepository provides data access for analytics summaries and
// on-the-fly aggregation against the crises collection.
type AnalyticsRepository struct {
	*mongoRepo.MongoRepository[model.AnalyticsSummary]
	crisesCollection *mongo.Collection
}

// NewAnalyticsRepository constructs an AnalyticsRepository.
func NewAnalyticsRepository(db *mongo.Database, crisesCollection *mongo.Collection) *AnalyticsRepository {
	return &AnalyticsRepository{
		MongoRepository:  mongoRepo.NewMongoRepository[model.AnalyticsSummary](db, "analytics"),
		crisesCollection: crisesCollection,
	}
}

// FindSummary looks up a pre-computed summary by region and period.
func (r *AnalyticsRepository) FindSummary(ctx context.Context, region, period string) (*model.AnalyticsSummary, error) {
	filter := bson.D{
		{Key: "region", Value: region},
		{Key: "period", Value: period},
	}
	result, err := r.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("AnalyticsRepository.FindSummary: %w", err)
	}
	return result, nil
}

// AggregateTrend queries the crises collection for counts grouped by day,
// optionally filtered by event_type.
func (r *AnalyticsRepository) AggregateTrend(ctx context.Context, eventType string, days int) ([]bson.M, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	matchFilter := bson.D{{Key: "start_time", Value: bson.D{{Key: "$gte", Value: since}}}}
	if eventType != "" {
		matchFilter = append(matchFilter, bson.E{Key: "event", Value: eventType})
	}
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: matchFilter}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: "%Y-%m-%d"},
				{Key: "date", Value: "$start_time"},
			}}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "avg_severity", Value: bson.D{{Key: "$avg", Value: "$severity"}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	cursor, err := r.crisesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("AnalyticsRepository.AggregateTrend: %w", err)
	}
	defer cursor.Close(ctx)
	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("AnalyticsRepository.AggregateTrend decode: %w", err)
	}
	return results, nil
}

// AggregateHeatmap queries crises grouped by a rounded lat/lng grid
// within the specified radius.
func (r *AnalyticsRepository) AggregateHeatmap(ctx context.Context, lat, lng, radiusKm float64) ([]bson.M, error) {
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
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "lat", Value: bson.D{{Key: "$round", Value: bson.A{"$lat", 2}}}},
				{Key: "lng", Value: bson.D{{Key: "$round", Value: bson.A{"$lng", 2}}}},
			}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "avg_severity", Value: bson.D{{Key: "$avg", Value: "$severity"}}},
		}}},
	}
	cursor, err := r.crisesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("AnalyticsRepository.AggregateHeatmap: %w", err)
	}
	defer cursor.Close(ctx)
	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("AnalyticsRepository.AggregateHeatmap decode: %w", err)
	}
	return results, nil
}
