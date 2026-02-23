package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// Subscription represents a user's alert subscription for a geographic area.
// When an alert is published near the subscription's Location within RadiusKm,
// the user is notified via the configured channel (WebSocket, push, etc.).
type Subscription struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id"       json:"user_id"`
	Location    geo.GeoJSONPoint   `bson:"location"      json:"location"`
	RadiusKm    float64            `bson:"radius_km"     json:"radius_km"`
	CrisisTypes []string           `bson:"crisis_types"  json:"crisis_types"` // empty = all types
	Active      bool               `bson:"active"        json:"active"`
	CreatedAt   time.Time          `bson:"created_at"    json:"created_at"`
}
