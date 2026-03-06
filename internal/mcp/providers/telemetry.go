package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"observability-hub/internal/telemetry"
)

// TelemetryProvider manages connections to Thanos and exposes telemetry tools.
type TelemetryProvider struct {
	thanosURL  string
	httpClient *http.Client
}

// NewTelemetryProvider creates a new telemetry provider connected to Thanos.
func NewTelemetryProvider(thanosURL string) *TelemetryProvider {
	telemetry.Info("creating new telemetry provider", "thanos_url", thanosURL)
	return &TelemetryProvider{
		thanosURL: thanosURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QueryMetrics executes a PromQL query against Thanos.
// Returns raw Prometheus API response (query result).
func (tp *TelemetryProvider) QueryMetrics(ctx context.Context, query string) (interface{}, error) {
	if query == "" {
		telemetry.Error("query metrics called with empty query")
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Validate query length to prevent abuse
	if len(query) > 10000 {
		telemetry.Warn("query exceeds max length", "query_len", len(query))
		return nil, fmt.Errorf("query too long (max 10000 chars)")
	}

	// Build URL with query parameters
	endpoint := fmt.Sprintf("%s/api/v1/query", tp.thanosURL)
	params := url.Values{}
	params.Add("query", query)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		telemetry.Error("failed to create request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	telemetry.Info("executing PromQL query", "query", query[:min(len(query), 100)])
	resp, err := tp.httpClient.Do(req)
	if err != nil {
		telemetry.Error("failed to query Thanos", "error", err)
		return nil, fmt.Errorf("failed to query Thanos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		telemetry.Error("Thanos returned non-OK status", "status", resp.StatusCode)
		return nil, fmt.Errorf("Thanos returned status %d", resp.StatusCode)
	}

	// Return raw body for now; in production, parse JSON and structure response
	var result map[string]interface{}
	if err := parseJSONResponse(resp, &result); err != nil {
		telemetry.Error("failed to parse response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	telemetry.Info("query executed successfully")
	return result, nil
}

// QueryLogs executes a LogQL query against Loki (no-op for now).
func (tp *TelemetryProvider) QueryLogs(ctx context.Context, query string) (interface{}, error) {
	telemetry.Info("QueryLogs called (no-op)", "query", query[:min(len(query), 50)])
	return map[string]interface{}{
		"status":  "not_implemented",
		"message": "hello mcp logs - LogQL queries coming soon",
		"query":   query,
	}, nil
}

// QueryTraces retrieves a trace by ID from Tempo (no-op for now).
func (tp *TelemetryProvider) QueryTraces(ctx context.Context, traceID string) (interface{}, error) {
	telemetry.Info("QueryTraces called (no-op)", "trace_id", traceID)
	return map[string]interface{}{
		"status":   "not_implemented",
		"message":  "hello mcp traces - Distributed trace queries coming soon",
		"trace_id": traceID,
	}, nil
}

// Close closes the provider's HTTP client and resources.
func (tp *TelemetryProvider) Close() error {
	telemetry.Info("closing telemetry provider")
	tp.httpClient.CloseIdleConnections()
	return nil
}

// parseJSONResponse decodes a JSON HTTP response body into v.
func parseJSONResponse(resp *http.Response, v interface{}) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

// min is a helper to get minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
