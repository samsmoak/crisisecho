package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// SavedLocation is an embedded struct within User for named saved locations.
type SavedLocation struct {
	Label    string           `bson:"label"     json:"label"`
	Location geo.GeoJSONPoint `bson:"location"  json:"location"`
	Lat      float64          `bson:"lat"       json:"lat"`
	Lng      float64          `bson:"lng"       json:"lng"`
	RadiusKm float64          `bson:"radius_km" json:"radius_km"`
}

// User represents a registered CrisisEcho user. Created automatically on first
// successful Firebase sign-in via the GET /api/users/me upsert flow.
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"    json:"id"`
	FirebaseUID    string             `bson:"firebase_uid"     json:"firebase_uid"`
	Email          string             `bson:"email"            json:"email"`
	Name           string             `bson:"name"             json:"name"`
	Picture        string             `bson:"picture"          json:"picture"`
	Role           string             `bson:"role"             json:"role"`
	SavedLocations []SavedLocation    `bson:"saved_locations"  json:"saved_locations"`
	CreatedAt      time.Time          `bson:"created_at"       json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at"       json:"updated_at"`
}
