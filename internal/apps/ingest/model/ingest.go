package model

import (
	"encoding/json"
	"time"
)

// Kafka topic names for all ingestion channels.
const (
	TopicSocialRaw      = "social_raw"
	TopicOfficialAlerts = "official_alerts"
	TopicNewsFeed       = "news_feed"
)

// KafkaMessage is the envelope for every message produced by Python ingestion workers.
// Payload is json.RawMessage so it serializes as a JSON sub-object (not base64),
// allowing the Go consumer to unmarshal it into a map without double-encoding.
type KafkaMessage struct {
	Topic      string          `json:"topic"`
	Source     string          `json:"source"`
	Payload    json.RawMessage `json:"payload"`
	ReceivedAt time.Time       `json:"received_at"`
}
