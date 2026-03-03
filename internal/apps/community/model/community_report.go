package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// CommunityReport is a user-submitted crisis report from the mobile app.
// MediaURLs contains S3 URLs uploaded via the /api/upload endpoint.
type CommunityReport struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"      json:"id"`
	UserID           string             `bson:"user_id"            json:"user_id"`
	EventType        string             `bson:"event_type"         json:"event_type"`
	Description      string             `bson:"description"        json:"description"`
	Location         geo.GeoJSONPoint   `bson:"location"           json:"location"`
	Lat              float64            `bson:"lat"                json:"lat"`
	Lng              float64            `bson:"lng"                json:"lng"`
	MediaURLs        []string           `bson:"media_urls"         json:"media_urls"`
	SeverityEstimate string             `bson:"severity_estimate"  json:"severity_estimate"`
	PeopleAffected   string             `bson:"people_affected"    json:"people_affected"`
	Status           string             `bson:"status"             json:"status"`
	CreatedAt        time.Time          `bson:"created_at"         json:"created_at"`
}
