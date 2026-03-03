package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/geo"
)

// EmergencyContact is an embedded struct within SOSProfile.
type EmergencyContact struct {
	Name  string `bson:"name"  json:"name"`
	Phone string `bson:"phone" json:"phone"`
}

// SOSProfile is a pre-configured SOS template the user sets up in advance.
// Up to 4 profiles per user. Triggered via POST /api/sos/trigger/:profileId.
type SOSProfile struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"       json:"id"`
	UserID            string             `bson:"user_id"             json:"user_id"`
	Label             string             `bson:"label"               json:"label"`
	EventType         string             `bson:"event_type"          json:"event_type"`
	Severity          int                `bson:"severity"            json:"severity"`
	MessageTemplate   string             `bson:"message_template"    json:"message_template"`
	EmergencyContacts []EmergencyContact `bson:"emergency_contacts"  json:"emergency_contacts"`
	Active            bool               `bson:"active"              json:"active"`
	CreatedAt         time.Time          `bson:"created_at"          json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at"          json:"updated_at"`
}

// SOSAlert is created when a user triggers an SOS profile.
// Published to Redis alerts:live with alert_kind="sos".
type SOSAlert struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"     json:"id"`
	UserID          string             `bson:"user_id"           json:"user_id"`
	SOSProfileID    primitive.ObjectID `bson:"sos_profile_id"    json:"sos_profile_id"`
	Label           string             `bson:"label"             json:"label"`
	EventType       string             `bson:"event_type"        json:"event_type"`
	Location        geo.GeoJSONPoint   `bson:"location"          json:"location"`
	Lat             float64            `bson:"lat"               json:"lat"`
	Lng             float64            `bson:"lng"               json:"lng"`
	MessageTemplate string             `bson:"message_template"  json:"message_template"`
	Severity        int                `bson:"severity"          json:"severity"`
	Status          string             `bson:"status"            json:"status"`
	TriggeredAt     time.Time          `bson:"triggered_at"      json:"triggered_at"`
	ResolvedAt      *time.Time         `bson:"resolved_at"       json:"resolved_at"`
}
