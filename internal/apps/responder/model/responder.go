package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// Responder is a community volunteer profile.
type Responder struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	UserID        string             `bson:"user_id"         json:"user_id"`
	Capabilities  []string           `bson:"capabilities"    json:"capabilities"`
	Availability  string             `bson:"availability"    json:"availability"`
	RadiusKm      float64            `bson:"radius_km"       json:"radius_km"`
	Location      geo.GeoJSONPoint   `bson:"location"        json:"location"`
	Lat           float64            `bson:"lat"             json:"lat"`
	Lng           float64            `bson:"lng"             json:"lng"`
	ResponseCount int                `bson:"response_count"  json:"response_count"`
	Rating        float64            `bson:"rating"          json:"rating"`
	Verified      bool               `bson:"verified"        json:"verified"`
	Active        bool               `bson:"active"          json:"active"`
	CreatedAt     time.Time          `bson:"created_at"      json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"      json:"updated_at"`
}

// Response records a responder's action on an SOS alert or crisis alert.
type Response struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"  json:"id"`
	ResponderID primitive.ObjectID `bson:"responder_id"   json:"responder_id"`
	AlertID     primitive.ObjectID `bson:"alert_id"       json:"alert_id"`
	AlertKind   string             `bson:"alert_kind"     json:"alert_kind"`
	Status      string             `bson:"status"         json:"status"`
	StartedAt   *time.Time         `bson:"started_at"     json:"started_at"`
	CompletedAt *time.Time         `bson:"completed_at"   json:"completed_at"`
	Rating      int                `bson:"rating"         json:"rating"`
	Notes       string             `bson:"notes"          json:"notes"`
}
