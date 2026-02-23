package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// RawPost is a document ingested from a source platform before preprocessing.
// Stored in per-source collections: twitter_posts, reddit_posts, etc.
// The User field is stored as a SHA-256 hash of the original username.
type RawPost struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"       json:"id"`
	Source             string             `bson:"source"              json:"source"`
	PostID             string             `bson:"post_id"             json:"post_id"`
	Text               string             `bson:"text"                json:"text"`
	User               string             `bson:"user"                json:"user"`               // SHA-256 hash
	Timestamp          time.Time          `bson:"timestamp"           json:"timestamp"`
	URL                string             `bson:"url"                 json:"url"`
	ImageURLs          []string           `bson:"image_urls"          json:"image_urls"`
	Location           geo.GeoJSONPoint   `bson:"location"            json:"location"`
	LocationConfidence float64            `bson:"location_confidence" json:"location_confidence"`
	LocationSource     string             `bson:"location_source"     json:"location_source"`    // gps|carmen|ner|geo_prior|unresolved
	IsRelevant         bool               `bson:"is_relevant"         json:"is_relevant"`
	CrisisType         string             `bson:"crisis_type"         json:"crisis_type"`
	Metadata           bson.M             `bson:"metadata"            json:"metadata"`
}

// UnifiedPost is a normalized, deduplicated post stored in the "posts" collection.
// Created by the Python preprocessing pipeline after cleaning, geocoding, and embedding.
// ClusterID is a pointer so it serializes as null (not omitted) when unset.
type UnifiedPost struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty"        json:"id"`
	Source             string              `bson:"source"               json:"source"`
	PostID             string              `bson:"post_id"              json:"post_id"`
	Text               string              `bson:"text"                 json:"text"`
	CleanedText        string              `bson:"cleaned_text"         json:"cleaned_text"`
	User               string              `bson:"user"                 json:"user"`
	Timestamp          time.Time           `bson:"timestamp"            json:"timestamp"`
	URL                string              `bson:"url"                  json:"url"`
	ImageURLs          []string            `bson:"image_urls"           json:"image_urls"`
	Location           geo.GeoJSONPoint    `bson:"location"             json:"location"`
	LocationConfidence float64             `bson:"location_confidence"  json:"location_confidence"`
	LocationSource     string              `bson:"location_source"      json:"location_source"`
	IsRelevant         bool                `bson:"is_relevant"          json:"is_relevant"`
	CrisisType         string              `bson:"crisis_type"          json:"crisis_type"`
	SeverityScore      int                 `bson:"severity_score"       json:"severity_score"`
	TextEmbeddingID    string              `bson:"text_embedding_id"    json:"text_embedding_id"`
	ImageEmbeddingIDs  []string            `bson:"image_embedding_ids"  json:"image_embedding_ids"`
	ClusterID          *primitive.ObjectID `bson:"cluster_id"           json:"cluster_id"`
	Metadata           bson.M              `bson:"metadata"             json:"metadata"` // includes username_hash
}
