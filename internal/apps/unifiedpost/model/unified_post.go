package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// UnifiedPost is a system-generated synthesis document representing what a cluster
// of SourcePosts is collectively describing. Created by the LLM agent after a cluster
// is formed. Stored in the "unified_posts" collection.
// The crisis document is only created if this UnifiedPost is verified.
type UnifiedPost struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty"           json:"id"`
	ClusterID             primitive.ObjectID `bson:"cluster_id"              json:"cluster_id"`
	EventType             string             `bson:"event_type"              json:"event_type"`
	Summary               string             `bson:"summary"                 json:"summary"`
	Location              geo.GeoJSONPoint   `bson:"location"                json:"location"`
	Lat                   float64            `bson:"lat"                     json:"lat"`
	Lng                   float64            `bson:"lng"                     json:"lng"`
	Severity              int                `bson:"severity"                json:"severity"`
	ConfidenceScore       float64            `bson:"confidence_score"        json:"confidence_score"`
	Sources               []string           `bson:"sources"                 json:"sources"`
	ContributorCount      int                `bson:"contributor_count"       json:"contributor_count"`
	OfficialCorroboration bool               `bson:"official_corroboration"  json:"official_corroboration"`
	PostIDs               []string           `bson:"post_ids"                json:"post_ids"`
	Verified              bool               `bson:"verified"                json:"verified"`
	CreatedAt             time.Time          `bson:"created_at"              json:"created_at"`
	UpdatedAt             time.Time          `bson:"updated_at"              json:"updated_at"`
}
