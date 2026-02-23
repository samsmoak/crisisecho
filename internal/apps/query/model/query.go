package model

import (
	clusterModel "crisisecho/internal/apps/cluster/model"
)

// QueryRequest is the parsed input for a natural-language crisis query.
type QueryRequest struct {
	Text   string  `json:"text"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Radius float64 `json:"radius"` // kilometres; informational, passed to Python sidecar
}

// QueryResponse is the structured response from the Python digest chain.
type QueryResponse struct {
	Digest   string                    `json:"digest"`
	Clusters []*clusterModel.Cluster   `json:"clusters"`
}
