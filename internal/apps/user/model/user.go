package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a registered CrisisEcho user. Created automatically on first
// successful Firebase sign-in via the GET /api/users/me upsert flow.
type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"    json:"id"`
	FirebaseUID string             `bson:"firebase_uid"     json:"firebase_uid"`
	Email       string             `bson:"email"            json:"email"`
	Name        string             `bson:"name"             json:"name"`
	Picture     string             `bson:"picture"          json:"picture"`
	Role        string             `bson:"role"             json:"role"`
	CreatedAt   time.Time          `bson:"created_at"       json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"       json:"updated_at"`
}
