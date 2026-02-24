package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/post/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// ─── RawPostRepository ────────────────────────────────────────────────────────

// RawPostRepository provides data access for a single source's raw post collection.
// Each source platform has its own collection (twitter_posts, reddit_posts, etc.).
type RawPostRepository struct {
	*mongoRepo.MongoRepository[model.RawPost]
}

// NewRawPostRepository constructs a RawPostRepository backed by the named collection.
// collectionName should come from database.CollectionNameForSource(source).
func NewRawPostRepository(db *mongo.Database, collectionName string) *RawPostRepository {
	return &RawPostRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.RawPost](db, collectionName),
	}
}

// FindByLocation returns RawPosts within radiusKm of (lat, lng) using a $geoNear aggregation.
// Requires a 2dsphere index on the "location" field of the backing collection.
func (r *RawPostRepository) FindByLocation(ctx context.Context, lat, lng, radiusKm float64) ([]*model.RawPost, error) {
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
		return nil, fmt.Errorf("RawPostRepository.FindByLocation: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.RawPost
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("RawPostRepository.FindByLocation decode: %w", err)
	}
	return results, nil
}

// FindRecent returns RawPosts whose Timestamp is within the last `minutes` minutes.
func (r *RawPostRepository) FindRecent(ctx context.Context, minutes int) ([]*model.RawPost, error) {
	since := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	filter := bson.D{{Key: "timestamp", Value: bson.D{{Key: "$gte", Value: since}}}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("RawPostRepository.FindRecent: %w", err)
	}
	return results, nil
}

// FindByCrisisType returns all RawPosts with the given crisis_type.
func (r *RawPostRepository) FindByCrisisType(ctx context.Context, crisisType string) ([]*model.RawPost, error) {
	filter := bson.D{{Key: "crisis_type", Value: crisisType}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("RawPostRepository.FindByCrisisType: %w", err)
	}
	return results, nil
}

// ─── SourcePostRepository ────────────────────────────────────────────────────

// SourcePostRepository provides data access for the unified "posts" collection.
type SourcePostRepository struct {
	*mongoRepo.MongoRepository[model.SourcePost]
}

// NewSourcePostRepository constructs a SourcePostRepository backed by the "posts" collection.
func NewSourcePostRepository(db *mongo.Database) *SourcePostRepository {
	return &SourcePostRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.SourcePost](db, "posts"),
	}
}

// FindNear returns SourcePosts within radiusKm of (lat, lng) using a $geoNear aggregation.
// Requires a 2dsphere index on the "location" field of "posts".
func (r *SourcePostRepository) FindNear(ctx context.Context, lat, lng, radiusKm float64) ([]*model.SourcePost, error) {
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
		return nil, fmt.Errorf("SourcePostRepository.FindNear: %w", err)
	}
	defer cursor.Close(ctx)
	var results []*model.SourcePost
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("SourcePostRepository.FindNear decode: %w", err)
	}
	return results, nil
}

// FindRecentRelevant returns is_relevant=true SourcePosts within the last `minutes` minutes.
func (r *SourcePostRepository) FindRecentRelevant(ctx context.Context, minutes int) ([]*model.SourcePost, error) {
	since := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	filter := bson.D{
		{Key: "timestamp",   Value: bson.D{{Key: "$gte", Value: since}}},
		{Key: "is_relevant", Value: true},
	}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("SourcePostRepository.FindRecentRelevant: %w", err)
	}
	return results, nil
}

// FindByClusterID returns all SourcePosts assigned to the given cluster.
func (r *SourcePostRepository) FindByClusterID(ctx context.Context, clusterID primitive.ObjectID) ([]*model.SourcePost, error) {
	filter := bson.D{{Key: "cluster_id", Value: clusterID}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("SourcePostRepository.FindByClusterID: %w", err)
	}
	return results, nil
}

// UpdateClusterID sets the cluster_id on the post identified by postID (hex ObjectID string).
func (r *SourcePostRepository) UpdateClusterID(ctx context.Context, postID string, clusterID primitive.ObjectID) error {
	oid, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return fmt.Errorf("SourcePostRepository.UpdateClusterID: invalid postID %q: %w", postID, err)
	}
	filter := bson.D{{Key: "_id", Value: oid}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "cluster_id", Value: clusterID}}}}
	res, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("SourcePostRepository.UpdateClusterID: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("SourcePostRepository.UpdateClusterID: post not found")
	}
	return nil
}
