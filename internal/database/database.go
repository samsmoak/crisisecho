package database

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ─── Connection helpers ───────────────────────────────────────────────────────

// ConnectMain opens a connection to the main MongoDB database.
// Reads MONGO_URI from the environment.
func ConnectMain(ctx context.Context) (*mongo.Client, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return nil, fmt.Errorf("database.ConnectMain: MONGO_URI is not set")
	}
	return connect(ctx, uri, "main")
}

// ConnectLocation opens a connection to the location MongoDB database.
// Reads MONGO_LOCATION_URI from the environment.
func ConnectLocation(ctx context.Context) (*mongo.Client, error) {
	uri := os.Getenv("MONGO_LOCATION_URI")
	if uri == "" {
		return nil, fmt.Errorf("database.ConnectLocation: MONGO_LOCATION_URI is not set")
	}
	return connect(ctx, uri, "location")
}

// ConnectVector opens a connection to the vector MongoDB database.
// Reads MONGO_VECTOR_URI from the environment.
func ConnectVector(ctx context.Context) (*mongo.Client, error) {
	uri := os.Getenv("MONGO_VECTOR_URI")
	if uri == "" {
		return nil, fmt.Errorf("database.ConnectVector: MONGO_VECTOR_URI is not set")
	}
	return connect(ctx, uri, "vector")
}

// connect dials MongoDB, pings the deployment, and returns the client.
func connect(ctx context.Context, uri, label string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("database.connect (%s): %w", label, err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("database.connect (%s) ping: %w", label, err)
	}
	return client, nil
}

// ─── Main DB collection helpers ───────────────────────────────────────────────

func mainDB(client *mongo.Client) *mongo.Database {
	return client.Database(os.Getenv("MONGO_DB_DATABASE"))
}

// GetPostsCollection returns the posts collection from the main database.
func GetPostsCollection(client *mongo.Client) *mongo.Collection {
	return mainDB(client).Collection("posts")
}

// GetClustersCollection returns the clusters collection from the main database.
func GetClustersCollection(client *mongo.Client) *mongo.Collection {
	return mainDB(client).Collection("clusters")
}

// GetCrisesCollection returns the crises collection from the main database.
func GetCrisesCollection(client *mongo.Client) *mongo.Collection {
	return mainDB(client).Collection("crises")
}

// GetAlertsCollection returns the alerts collection from the main database.
func GetAlertsCollection(client *mongo.Client) *mongo.Collection {
	return mainDB(client).Collection("alerts")
}

// GetSubscriptionsCollection returns the subscriptions collection from the main database.
func GetSubscriptionsCollection(client *mongo.Client) *mongo.Collection {
	return mainDB(client).Collection("subscriptions")
}

// GetOfficialAlertsCollection returns the official_alerts collection from the main database.
func GetOfficialAlertsCollection(client *mongo.Client) *mongo.Collection {
	return mainDB(client).Collection("official_alerts")
}

// sourceCollectionNames maps source platform names to their MongoDB collection names.
var sourceCollectionNames = map[string]string{
	"twitter":    "twitter_posts",
	"reddit":     "reddit_posts",
	"bluesky":    "bluesky_posts",
	"mastodon":   "mastodon_posts",
	"nextdoor":   "nextdoor_posts",
	"telegram":   "telegram_posts",
	"nws":        "nws_alerts",
	"usgs":       "usgs_alerts",
	"gdelt":      "gdelt_posts",
	"patch":      "patch_posts",
	"pulsepoint": "pulsepoint_posts",
}

// CollectionNameForSource returns the MongoDB collection name for the given source platform.
// Unknown sources fall back to "<source>_posts". This is a pure function with no DB side effects.
func CollectionNameForSource(source string) string {
	if name, ok := sourceCollectionNames[source]; ok {
		return name
	}
	return source + "_posts"
}

// GetSourceCollection returns the raw ingestion collection for the given source name.
// Source names map to collection names: "twitter" → "twitter_posts", etc.
// Unknown sources fall back to a "<source>_posts" naming convention.
func GetSourceCollection(client *mongo.Client, source string) *mongo.Collection {
	return mainDB(client).Collection(CollectionNameForSource(source))
}

// ─── Location DB collection helpers ──────────────────────────────────────────

func locationDB(client *mongo.Client) *mongo.Database {
	return client.Database(os.Getenv("MONGO_LOCATION_DB_DATABASE"))
}

// GetGeoPriorsCollection returns the geo_priors collection from the location database.
func GetGeoPriorsCollection(client *mongo.Client) *mongo.Collection {
	return locationDB(client).Collection("geo_priors")
}

// GetPlaceIndexCollection returns the place_index collection from the location database.
func GetPlaceIndexCollection(client *mongo.Client) *mongo.Collection {
	return locationDB(client).Collection("place_index")
}

// GetLocationCacheCollection returns the location_cache collection from the location database.
func GetLocationCacheCollection(client *mongo.Client) *mongo.Collection {
	return locationDB(client).Collection("location_cache")
}

// ─── Vector DB collection helpers ────────────────────────────────────────────

func vectorDB(client *mongo.Client) *mongo.Database {
	return client.Database(os.Getenv("MONGO_VECTOR_DB_DATABASE"))
}

// GetTextEmbeddingsCollection returns the text_embeddings collection from the vector database.
func GetTextEmbeddingsCollection(client *mongo.Client) *mongo.Collection {
	return vectorDB(client).Collection("text_embeddings")
}

// GetImageEmbeddingsCollection returns the image_embeddings collection from the vector database.
func GetImageEmbeddingsCollection(client *mongo.Client) *mongo.Collection {
	return vectorDB(client).Collection("image_embeddings")
}
