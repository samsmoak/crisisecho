package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	vectorRepo "crisisecho/internal/database/abstractrepository/vectordb"
)

// RAGService defines the public contract for the RAG pipeline wrapper.
type RAGService interface {
	// TriggerPipeline asks the Python sidecar to run the full retrieval→agent
	// pipeline for the given coordinates.
	TriggerPipeline(ctx context.Context, lat, lng float64) error

	// RunQuery asks the Python sidecar's digest chain to answer a natural-
	// language question about active clusters near the given coordinates.
	RunQuery(ctx context.Context, query string, lat, lng float64) (string, error)
}

type ragService struct {
	sidecarURL string
	httpClient *http.Client
	vectorRepo *vectorRepo.VectorRepository // reserved for Go-side vector lookups (Prompt 3)
}

const maxRetries = 3

// NewRAGService constructs a RAGService.
// sidecarURL is read from $PYTHON_SIDECAR_URL (default http://localhost:8081).
// vr is the VectorRepository injected for any Go-side vector lookups.
func NewRAGService(vr *vectorRepo.VectorRepository) RAGService {
	url := os.Getenv("PYTHON_SIDECAR_URL")
	if url == "" {
		url = "http://localhost:8081"
	}
	return &ragService{
		sidecarURL: url,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		vectorRepo: vr,
	}
}

// TriggerPipeline posts a trigger request to the Python sidecar with
// up to maxRetries attempts and exponential backoff.
func (s *ragService) TriggerPipeline(ctx context.Context, lat, lng float64) error {
	body, err := json.Marshal(map[string]float64{"lat": lat, "lng": lng})
	if err != nil {
		return fmt.Errorf("RAGService.TriggerPipeline: marshal: %w", err)
	}

	return s.callWithRetry(ctx, http.MethodPost, "/internal/pipeline/trigger", body, nil)
}

// RunQuery posts to the Python sidecar's /internal/query endpoint and returns
// only the digest string.
func (s *ragService) RunQuery(ctx context.Context, query string, lat, lng float64) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"text": query,
		"lat":  lat,
		"lng":  lng,
	})
	if err != nil {
		return "", fmt.Errorf("RAGService.RunQuery: marshal: %w", err)
	}

	var result struct {
		Digest string `json:"digest"`
	}
	if err := s.callWithRetry(ctx, http.MethodPost, "/internal/query", body, &result); err != nil {
		return "", err
	}
	return result.Digest, nil
}

// ── HTTP helper ───────────────────────────────────────────────────────────────

// callWithRetry executes an HTTP request with up to maxRetries attempts,
// doubling the wait on each failure. Non-2xx responses are treated as errors.
// If dst is non-nil, the response body is decoded into it.
func (s *ragService) callWithRetry(
	ctx context.Context,
	method, path string,
	body []byte,
	dst interface{},
) error {
	var lastErr error
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method,
			s.sidecarURL+path,
			bytes.NewReader(body),
		)
		if err != nil {
			return fmt.Errorf("RAGService: new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("RAGService: attempt %d/%d failed: %v", attempt, maxRetries, err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if dst != nil {
					if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
						return fmt.Errorf("RAGService: decode: %w", err)
					}
				}
				return nil
			}
			raw, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("sidecar returned %d: %s", resp.StatusCode, string(raw))
			log.Printf("RAGService: attempt %d/%d: %v", attempt, maxRetries, lastErr)
		}

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
	}
	return fmt.Errorf("RAGService.callWithRetry: all %d attempts failed: %w", maxRetries, lastErr)
}
