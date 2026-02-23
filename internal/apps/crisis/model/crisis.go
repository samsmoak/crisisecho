package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// Crisis is a confirmed or suspected crisis event derived from one or more clusters.
// Lat and Lng are stored as flat fields alongside Location for query convenience.
type Crisis struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Event       string             `bson:"event"         json:"event"`
	Location    geo.GeoJSONPoint   `bson:"location"      json:"location"`
	Lat         float64            `bson:"lat"           json:"lat"`
	Lng         float64            `bson:"lng"           json:"lng"`
	Severity    int                `bson:"severity"      json:"severity"`
	Confirmed   bool               `bson:"confirmed"     json:"confirmed"`
	Sources     []string           `bson:"sources"       json:"sources"`
	Description string             `bson:"description"   json:"description"`
	ImageURLs   []string           `bson:"image_urls"    json:"image_urls"`
	StartTime   time.Time          `bson:"start_time"    json:"start_time"`
	LastUpdated time.Time          `bson:"last_updated"  json:"last_updated"`
}
