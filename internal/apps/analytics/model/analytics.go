package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AnalyticsSummary is a pre-computed or on-the-fly analytics document
// for a geographic region over a time period.
type AnalyticsSummary struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty"               json:"id"`
	Region                 string             `bson:"region"                      json:"region"`
	Period                 string             `bson:"period"                      json:"period"`
	PeriodStart            time.Time          `bson:"period_start"                json:"period_start"`
	PeriodEnd              time.Time          `bson:"period_end"                  json:"period_end"`
	TotalCrises            int                `bson:"total_crises"                json:"total_crises"`
	TotalAlerts            int                `bson:"total_alerts"                json:"total_alerts"`
	TotalSOSAlerts         int                `bson:"total_sos_alerts"            json:"total_sos_alerts"`
	EventTypeBreakdown     map[string]int     `bson:"event_type_breakdown"        json:"event_type_breakdown"`
	AvgSeverity            float64            `bson:"avg_severity"                json:"avg_severity"`
	AvgResponseTimeMinutes float64            `bson:"avg_response_time_minutes"   json:"avg_response_time_minutes"`
	TopSources             []string           `bson:"top_sources"                 json:"top_sources"`
	CreatedAt              time.Time          `bson:"created_at"                  json:"created_at"`
}
