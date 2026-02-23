package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/cluster/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// ClusterRepository provides data access for the clusters collection.
type ClusterRepository struct {
	*mongoRepo.MongoRepository[model.Cluster]
}

// NewClusterRepository constructs a ClusterRepository backed by the "clusters" collection.
func NewClusterRepository(db *mongo.Database) *ClusterRepository {
	return &ClusterRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.Cluster](db, "clusters"),
	}
}

// FindActiveNear returns active clusters within radiusKm of (lat, lng).
// Uses $geoNear with a pre-filter on status=active for efficiency.
// Requires a 2dsphere index on the "centroid" field.
func (r *ClusterRepository) FindActiveNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.Cluster, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$geoNear", Value: bson.D{
			{Key: "near", Value: bson.D{
				{Key: "type", Value: "Point"},
				{Key: "coordinates", Value: bson.A{lng, lat}},
			}},
			{Key: "distanceField", Value: "distance_m"},
			{Key: "maxDistance", Value: radiusKm * 1000},
			{Key: "spherical", Value: true},
			{Key: "key", Value: "centroid"},
			{Key: "query", Value: bson.D{{Key: "status", Value: model.StatusActive}}},
		}}},
	}
	cursor, err := r.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("ClusterRepository.FindActiveNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.Cluster
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("ClusterRepository.FindActiveNear decode: %w", err)
	}
	return results, nil
}

// FindByStatus returns all clusters with the given status.
func (r *ClusterRepository) FindByStatus(ctx context.Context, status string) ([]*model.Cluster, error) {
	filter := bson.D{{Key: "status", Value: status}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("ClusterRepository.FindByStatus: %w", err)
	}
	return results, nil
}

// UpdateStatus sets the status and last_updated fields on the given cluster.
func (r *ClusterRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "status",       Value: status},
		{Key: "last_updated", Value: time.Now().UTC()},
	}}}
	res, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("ClusterRepository.UpdateStatus: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("ClusterRepository.UpdateStatus: cluster not found")
	}
	return nil
}
