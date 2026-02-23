package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// Cluster status values.
const (
	StatusActive     = "active"
	StatusResolved   = "resolved"
	StatusMonitoring = "monitoring"
)

// Cluster is a group of geographically and temporally proximate posts
// about the same crisis event. Created and updated by the clustering service.
type Cluster struct {
	ID                    primitive.ObjectID   `bson:"_id,omitempty"          json:"id"`
	Centroid              geo.GeoJSONPoint     `bson:"centroid"               json:"centroid"`
	AffectedArea          geo.GeoJSONPolygon   `bson:"affected_area"          json:"affected_area"`
	CrisisType            string               `bson:"crisis_type"            json:"crisis_type"`
	Severity              int                  `bson:"severity"               json:"severity"`
	SeverityRationale     string               `bson:"severity_rationale"     json:"severity_rationale"`
	Summary               string               `bson:"summary"                json:"summary"`
	PostCount             int                  `bson:"post_count"             json:"post_count"`
	ContributorCount      int                  `bson:"contributor_count"      json:"contributor_count"` // distinct user accounts
	PostIDs               []primitive.ObjectID `bson:"post_ids"               json:"post_ids"`
	Sources               []string             `bson:"sources"                json:"sources"`
	OfficialCorroboration bool                 `bson:"official_corroboration" json:"official_corroboration"`
	LocationConfidence    float64              `bson:"location_confidence"    json:"location_confidence"`
	StartTime             time.Time            `bson:"start_time"             json:"start_time"`
	LastUpdated           time.Time            `bson:"last_updated"           json:"last_updated"`
	Status                string               `bson:"status"                 json:"status"` // active|resolved|monitoring
}
