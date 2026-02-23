package locationRepo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LocationPrior is a known location reference used to infer coordinates from
// unresolved text mentions in posts (e.g. "near the park on 5th").
type LocationPrior struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Text           string             `bson:"text"           json:"text"`
	NormalizedText string             `bson:"normalized_text" json:"normalized_text"`
	Lat            float64            `bson:"lat"            json:"lat"`
	Lng            float64            `bson:"lng"            json:"lng"`
	Confidence     float64            `bson:"confidence"     json:"confidence"`
	Source         string             `bson:"source"         json:"source"` // carmen | manual | crowd
	HitCount       int                `bson:"hit_count"      json:"hit_count"`
	CreatedAt      int64              `bson:"created_at"     json:"created_at"`
	UpdatedAt      int64              `bson:"updated_at"     json:"updated_at"`
}

// CachedGeocode is a cached geocoding result keyed by the SHA-256 hash of the
// normalized input text. Prevents duplicate geocoding calls for identical inputs.
type CachedGeocode struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TextHash     string             `bson:"text_hash"     json:"text_hash"`
	OriginalText string             `bson:"original_text" json:"original_text"`
	Lat          float64            `bson:"lat"           json:"lat"`
	Lng          float64            `bson:"lng"           json:"lng"`
	Confidence   float64            `bson:"confidence"    json:"confidence"`
	Provider     string             `bson:"provider"      json:"provider"` // carmen | ner
	CachedAt     int64              `bson:"cached_at"     json:"cached_at"`
}

// PlaceEntry is a named geographic place (neighborhood, intersection, zip code,
// landmark) stored in the place index for fast geo lookups.
type PlaceEntry struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name"         json:"name"`
	Type        string             `bson:"type"         json:"type"` // neighborhood | intersection | zip | landmark
	ZipCode     string             `bson:"zip_code"     json:"zip_code"`
	Lat         float64            `bson:"lat"          json:"lat"`
	Lng         float64            `bson:"lng"          json:"lng"`
	Aliases     []string           `bson:"aliases"      json:"aliases"`
	BoundingBox [4]float64         `bson:"bounding_box" json:"bounding_box"` // [minLng, minLat, maxLng, maxLat]
	City        string             `bson:"city"         json:"city"`
	State       string             `bson:"state"        json:"state"`
	Country     string             `bson:"country"      json:"country"`
}

// LocationRepository provides access to all three location database collections.
// It is initialized once and injected into services that need geocoding support.
type LocationRepository struct {
	GeoPriors     *mongo.Collection
	PlaceIndex    *mongo.Collection
	LocationCache *mongo.Collection
}

// NewLocationRepository constructs a LocationRepository from the three pre-obtained
// collection references. Use database.Get*Collection helpers to build these.
func NewLocationRepository(geoPriors, placeIndex, locationCache *mongo.Collection) *LocationRepository {
	return &LocationRepository{
		GeoPriors:     geoPriors,
		PlaceIndex:    placeIndex,
		LocationCache: locationCache,
	}
}

// GetGeoPrior looks up a location prior by exact normalized text match.
// Returns nil, nil when no prior exists (not an error).
func (r *LocationRepository) GetGeoPrior(ctx context.Context, text string) (*LocationPrior, error) {
	filter := bson.D{{Key: "normalized_text", Value: text}}
	var prior LocationPrior
	if err := r.GeoPriors.FindOne(ctx, filter).Decode(&prior); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("LocationRepository.GetGeoPrior: %w", err)
	}
	return &prior, nil
}

// SaveGeoPrior upserts a location prior keyed by normalized_text. If a prior
// already exists for this text it is replaced; otherwise a new document is inserted.
func (r *LocationRepository) SaveGeoPrior(ctx context.Context, prior *LocationPrior) error {
	filter := bson.D{{Key: "normalized_text", Value: prior.NormalizedText}}
	update := bson.D{{Key: "$set", Value: prior}}
	opts := options.Update().SetUpsert(true)
	if _, err := r.GeoPriors.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("LocationRepository.SaveGeoPrior: %w", err)
	}
	return nil
}

// GetCachedGeocode retrieves a previously cached geocoding result by the
// SHA-256 hash of the normalized input text.
// Returns nil, nil when no cached result exists.
func (r *LocationRepository) GetCachedGeocode(ctx context.Context, textHash string) (*CachedGeocode, error) {
	filter := bson.D{{Key: "text_hash", Value: textHash}}
	var cache CachedGeocode
	if err := r.LocationCache.FindOne(ctx, filter).Decode(&cache); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("LocationRepository.GetCachedGeocode: %w", err)
	}
	return &cache, nil
}

// SaveCachedGeocode upserts a geocoding result keyed by text_hash.
func (r *LocationRepository) SaveCachedGeocode(ctx context.Context, cache *CachedGeocode) error {
	filter := bson.D{{Key: "text_hash", Value: cache.TextHash}}
	update := bson.D{{Key: "$set", Value: cache}}
	opts := options.Update().SetUpsert(true)
	if _, err := r.LocationCache.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("LocationRepository.SaveCachedGeocode: %w", err)
	}
	return nil
}

// FindNearbyPriors returns location priors whose coordinates fall within radiusM
// metres of the given lat/lng, ordered by ascending distance.
func (r *LocationRepository) FindNearbyPriors(ctx context.Context, lat, lng float64, radiusM int) ([]*LocationPrior, error) {
	// $geoNear requires a 2dsphere index on the location field.
	// We store lat/lng as flat fields rather than GeoJSON here for simplicity,
	// so we use $nearSphere with a legacy coordinate pair.
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$geoNear", Value: bson.D{
			{Key: "near", Value: bson.D{
				{Key: "type", Value: "Point"},
				{Key: "coordinates", Value: bson.A{lng, lat}},
			}},
			{Key: "distanceField", Value: "distance_m"},
			{Key: "maxDistance", Value: float64(radiusM)},
			{Key: "spherical", Value: true},
			{Key: "key", Value: "location"},
		}}},
	}

	cursor, err := r.GeoPriors.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("LocationRepository.FindNearbyPriors: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*LocationPrior
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("LocationRepository.FindNearbyPriors decode: %w", err)
	}
	return results, nil
}
