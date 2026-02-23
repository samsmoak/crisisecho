package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// Alert is a published crisis alert derived from a cluster that passed all thresholds.
// Published to MongoDB and the Redis "alerts:live" Pub/Sub channel.
type Alert struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"    json:"id"`
	ClusterID       primitive.ObjectID `bson:"cluster_id"       json:"cluster_id"`
	AlertText       string             `bson:"alert_text"       json:"alert_text"`
	Severity        int                `bson:"severity"         json:"severity"`
	CrisisType      string             `bson:"crisis_type"      json:"crisis_type"`
	Centroid        geo.GeoJSONPoint   `bson:"centroid"         json:"centroid"`
	PublishedAt     time.Time          `bson:"published_at"     json:"published_at"`
	SourcePlatforms []string           `bson:"source_platforms" json:"source_platforms"`
	NotifiedUsers   []string           `bson:"notified_users"   json:"notified_users"`
}
