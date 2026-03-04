package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// SavedLocation represents a user's named saved location stored in its own collection.
type SavedLocation struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id"       json:"user_id"`
	Label     string             `bson:"label"         json:"label"`
	Location  geo.GeoJSONPoint   `bson:"location"      json:"location"`
	Lat       float64            `bson:"lat"           json:"lat"`
	Lng       float64            `bson:"lng"           json:"lng"`
	RadiusKm  float64            `bson:"radius_km"     json:"radius_km"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"    json:"updated_at"`
}
