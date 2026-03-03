package model

import "time"

// CrisisFilter contains optional search parameters for filtering crises.
// Zero-value fields are ignored when building the query.
type CrisisFilter struct {
	EventType   string     `json:"event_type"`
	SeverityMin int        `json:"severity_min"`
	SeverityMax int        `json:"severity_max"`
	Since       *time.Time `json:"since"`
	Until       *time.Time `json:"until"`
	Confirmed   *bool      `json:"confirmed"`
}
