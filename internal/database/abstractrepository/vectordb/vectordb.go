package vectorRepo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TextEmbeddingDoc holds a 1024-dimensional Voyage AI embedding for a single
// processed post, together with metadata used as filter fields in $vectorSearch.
type TextEmbeddingDoc struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID        string             `bson:"post_id"        json:"post_id"`
	Source        string             `bson:"source"         json:"source"`
	Vector        []float64          `bson:"vector"         json:"vector"`         // 1024-dim
	Timestamp     int64              `bson:"timestamp"      json:"timestamp"`      // Unix seconds
	Lat           float64            `bson:"lat"            json:"lat"`
	Lng           float64            `bson:"lng"            json:"lng"`
	CrisisType    string             `bson:"crisis_type"    json:"crisis_type"`
	SeverityScore int                `bson:"severity_score" json:"severity_score"`
	IsRelevant    bool               `bson:"is_relevant"    json:"is_relevant"`
}

// ImageEmbeddingDoc holds a 512-dimensional CLIP embedding for a single post image.
type ImageEmbeddingDoc struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID    string             `bson:"post_id"   json:"post_id"`
	ImageURL  string             `bson:"image_url" json:"image_url"`
	Source    string             `bson:"source"    json:"source"`
	Vector    []float64          `bson:"vector"    json:"vector"`    // 512-dim
	Timestamp int64              `bson:"timestamp" json:"timestamp"` // Unix seconds
	Lat       float64            `bson:"lat"       json:"lat"`
	Lng       float64            `bson:"lng"       json:"lng"`
}

// VectorFilter contains optional metadata constraints applied as pre-filters
// inside the $vectorSearch aggregation stage.
type VectorFilter struct {
	MinTimestamp int64
	MaxTimestamp int64
	MinLat       float64
	MaxLat       float64
	MinLng       float64
	MaxLng       float64
	CrisisType   string
	IsRelevant   *bool // nil = no filter
}

// VectorRepository provides access to both vector collections in the vector
// database. Instantiated once and injected into services that need semantic search.
type VectorRepository struct {
	TextEmbeddings  *mongo.Collection
	ImageEmbeddings *mongo.Collection
}

// NewVectorRepository constructs a VectorRepository from the two pre-obtained
// collection references. Use database.Get*Collection helpers to build these.
func NewVectorRepository(textEmbeddings, imageEmbeddings *mongo.Collection) *VectorRepository {
	return &VectorRepository{
		TextEmbeddings:  textEmbeddings,
		ImageEmbeddings: imageEmbeddings,
	}
}

// UpsertTextVector inserts or replaces the text embedding document for a post.
// The upsert is keyed by post_id so each post has at most one text vector.
func (r *VectorRepository) UpsertTextVector(ctx context.Context, doc *TextEmbeddingDoc) error {
	filter := bson.D{{Key: "post_id", Value: doc.PostID}}
	update := bson.D{{Key: "$set", Value: doc}}
	opts := options.Update().SetUpsert(true)
	if _, err := r.TextEmbeddings.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("VectorRepository.UpsertTextVector: %w", err)
	}
	return nil
}

// UpsertImageVector inserts or replaces the image embedding document for a post.
// The upsert is keyed by post_id + image_url.
func (r *VectorRepository) UpsertImageVector(ctx context.Context, doc *ImageEmbeddingDoc) error {
	filter := bson.D{
		{Key: "post_id", Value: doc.PostID},
		{Key: "image_url", Value: doc.ImageURL},
	}
	update := bson.D{{Key: "$set", Value: doc}}
	opts := options.Update().SetUpsert(true)
	if _, err := r.ImageEmbeddings.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("VectorRepository.UpsertImageVector: %w", err)
	}
	return nil
}

// SearchTextVectors runs an Atlas $vectorSearch aggregation against text_embeddings.
// queryVector must be 1024-dimensional. filter narrows results by metadata fields.
func (r *VectorRepository) SearchTextVectors(ctx context.Context, queryVector []float64, limit int, filter VectorFilter) ([]*TextEmbeddingDoc, error) {
	preFilter := buildTextPreFilter(filter)

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$vectorSearch", Value: bson.D{
			{Key: "index", Value: "text_vector_index"},
			{Key: "path", Value: "vector"},
			{Key: "queryVector", Value: queryVector},
			{Key: "numCandidates", Value: 200},
			{Key: "limit", Value: limit},
			{Key: "filter", Value: preFilter},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "post_id", Value: 1},
			{Key: "source", Value: 1},
			{Key: "lat", Value: 1},
			{Key: "lng", Value: 1},
			{Key: "crisis_type", Value: 1},
			{Key: "severity_score", Value: 1},
			{Key: "is_relevant", Value: 1},
			{Key: "timestamp", Value: 1},
			{Key: "score", Value: bson.D{{Key: "$meta", Value: "vectorSearchScore"}}},
		}}},
	}

	cursor, err := r.TextEmbeddings.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("VectorRepository.SearchTextVectors: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*TextEmbeddingDoc
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("VectorRepository.SearchTextVectors decode: %w", err)
	}
	return results, nil
}

// SearchImageVectors runs an Atlas $vectorSearch aggregation against image_embeddings.
// queryVector must be 512-dimensional.
func (r *VectorRepository) SearchImageVectors(ctx context.Context, queryVector []float64, limit int, filter VectorFilter) ([]*ImageEmbeddingDoc, error) {
	preFilter := buildImagePreFilter(filter)

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$vectorSearch", Value: bson.D{
			{Key: "index", Value: "image_vector_index"},
			{Key: "path", Value: "vector"},
			{Key: "queryVector", Value: queryVector},
			{Key: "numCandidates", Value: 200},
			{Key: "limit", Value: limit},
			{Key: "filter", Value: preFilter},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "post_id", Value: 1},
			{Key: "image_url", Value: 1},
			{Key: "source", Value: 1},
			{Key: "lat", Value: 1},
			{Key: "lng", Value: 1},
			{Key: "timestamp", Value: 1},
			{Key: "score", Value: bson.D{{Key: "$meta", Value: "vectorSearchScore"}}},
		}}},
	}

	cursor, err := r.ImageEmbeddings.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("VectorRepository.SearchImageVectors: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ImageEmbeddingDoc
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("VectorRepository.SearchImageVectors decode: %w", err)
	}
	return results, nil
}

// DeleteTextVector removes the text embedding document for the given post_id.
func (r *VectorRepository) DeleteTextVector(ctx context.Context, postID string) error {
	filter := bson.D{{Key: "post_id", Value: postID}}
	if _, err := r.TextEmbeddings.DeleteOne(ctx, filter); err != nil {
		return fmt.Errorf("VectorRepository.DeleteTextVector: %w", err)
	}
	return nil
}

// DeleteOldVectors purges text and image embedding documents whose timestamp
// is older than olderThanUnix (Unix seconds). Used for TTL-style cleanup.
func (r *VectorRepository) DeleteOldVectors(ctx context.Context, olderThanUnix int64) error {
	filter := bson.D{{Key: "timestamp", Value: bson.D{{Key: "$lt", Value: olderThanUnix}}}}

	if _, err := r.TextEmbeddings.DeleteMany(ctx, filter); err != nil {
		return fmt.Errorf("VectorRepository.DeleteOldVectors (text): %w", err)
	}
	if _, err := r.ImageEmbeddings.DeleteMany(ctx, filter); err != nil {
		return fmt.Errorf("VectorRepository.DeleteOldVectors (image): %w", err)
	}
	return nil
}

// buildTextPreFilter constructs the $vectorSearch filter document for text embeddings.
// Only non-zero / non-nil fields in VectorFilter are included.
func buildTextPreFilter(f VectorFilter) bson.D {
	conditions := bson.D{}

	if f.MinTimestamp > 0 {
		conditions = append(conditions, bson.E{
			Key:   "timestamp",
			Value: bson.D{{Key: "$gt", Value: f.MinTimestamp}},
		})
	}
	if f.MinLat != 0 || f.MaxLat != 0 {
		conditions = append(conditions, bson.E{
			Key:   "lat",
			Value: bson.D{{Key: "$gte", Value: f.MinLat}, {Key: "$lte", Value: f.MaxLat}},
		})
	}
	if f.MinLng != 0 || f.MaxLng != 0 {
		conditions = append(conditions, bson.E{
			Key:   "lng",
			Value: bson.D{{Key: "$gte", Value: f.MinLng}, {Key: "$lte", Value: f.MaxLng}},
		})
	}
	if f.CrisisType != "" {
		conditions = append(conditions, bson.E{Key: "crisis_type", Value: f.CrisisType})
	}
	if f.IsRelevant != nil {
		conditions = append(conditions, bson.E{Key: "is_relevant", Value: *f.IsRelevant})
	}
	return conditions
}

// buildImagePreFilter constructs the $vectorSearch filter document for image embeddings.
func buildImagePreFilter(f VectorFilter) bson.D {
	conditions := bson.D{}

	if f.MinTimestamp > 0 {
		conditions = append(conditions, bson.E{
			Key:   "timestamp",
			Value: bson.D{{Key: "$gt", Value: f.MinTimestamp}},
		})
	}
	if f.MinLat != 0 || f.MaxLat != 0 {
		conditions = append(conditions, bson.E{
			Key:   "lat",
			Value: bson.D{{Key: "$gte", Value: f.MinLat}, {Key: "$lte", Value: f.MaxLat}},
		})
	}
	if f.MinLng != 0 || f.MaxLng != 0 {
		conditions = append(conditions, bson.E{
			Key:   "lng",
			Value: bson.D{{Key: "$gte", Value: f.MinLng}, {Key: "$lte", Value: f.MaxLng}},
		})
	}
	return conditions
}
