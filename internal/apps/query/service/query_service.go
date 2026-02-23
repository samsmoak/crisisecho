package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	queryModel "crisisecho/internal/apps/query/model"
)

// QueryService defines the public contract for the query domain.
type QueryService interface {
	RunQuery(ctx context.Context, req *queryModel.QueryRequest) (*queryModel.QueryResponse, error)
}

type queryService struct {
	sidecarURL string
	httpClient *http.Client
}

// NewQueryService constructs a QueryService that delegates to the Python sidecar.
// sidecarURL defaults to $PYTHON_SIDECAR_URL or http://localhost:8081.
func NewQueryService() QueryService {
	url := os.Getenv("PYTHON_SIDECAR_URL")
	if url == "" {
		url = "http://localhost:8081"
	}
	return &queryService{
		sidecarURL: url,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// RunQuery sends the query to the Python sidecar's /internal/query endpoint
// and returns the digest string plus the matching cluster list.
func (s *queryService) RunQuery(ctx context.Context, req *queryModel.QueryRequest) (*queryModel.QueryResponse, error) {
	body, err := json.Marshal(map[string]interface{}{
		"text": req.Text,
		"lat":  req.Lat,
		"lng":  req.Lng,
	})
	if err != nil {
		return nil, fmt.Errorf("QueryService.RunQuery: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.sidecarURL+"/internal/query",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("QueryService.RunQuery: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("QueryService.RunQuery: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("QueryService.RunQuery: sidecar returned %d: %s",
			resp.StatusCode, string(raw))
	}

	var result queryModel.QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("QueryService.RunQuery: decode: %w", err)
	}
	return &result, nil
}
