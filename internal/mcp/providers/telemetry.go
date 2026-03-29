package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"observability-hub/internal/telemetry"
)

// TelemetryProvider manages connections to Thanos, Loki, and Tempo and exposes telemetry tools.
type TelemetryProvider struct {
	thanosURL  string
	lokiURL    string
	tempoURL   string
	httpClient *http.Client
}

// NewTelemetryProvider creates a new telemetry provider connected to Thanos, Loki, and Tempo.
func NewTelemetryProvider(thanosURL, lokiURL, tempoURL string) *TelemetryProvider {
	return NewTelemetryProviderWithClient(thanosURL, lokiURL, tempoURL, nil)
}

// NewTelemetryProviderWithClient creates a new telemetry provider, allowing injection of a custom HTTP client.
// Passing a nil client will create a default client with a 30s timeout.
func NewTelemetryProviderWithClient(thanosURL, lokiURL, tempoURL string, client *http.Client) *TelemetryProvider {
	telemetry.Info("creating new telemetry provider", "thanos_url", thanosURL, "loki_url", lokiURL, "tempo_url", tempoURL)
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &TelemetryProvider{
		thanosURL:  thanosURL,
		lokiURL:    lokiURL,
		tempoURL:   tempoURL,
		httpClient: client,
	}
}

// QueryMetrics executes a PromQL query against Thanos.
// Returns raw Prometheus API response (query result).
//
// Limits:
//   - Uses instant query endpoint (/api/v1/query) — returns current value only, no time range.
//   - Time windows are expressed inline in PromQL (e.g. rate(...[24h])), not as a parameter.
//   - Query length: max 5,000 chars (our safety cap, not a Thanos limit).
//   - No result count cap — Thanos returns all matching series.
func (tp *TelemetryProvider) QueryMetrics(ctx context.Context, query string) (interface{}, error) {
	if query == "" {
		telemetry.Error("query metrics called with empty query")
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Validate query length to prevent abuse
	if len(query) > 5000 {
		telemetry.Warn("query exceeds max length", "query_len", len(query))
		return nil, fmt.Errorf("query too long (max 5000 chars)")
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

// QueryLogs executes a LogQL query against Loki and returns matching log streams.
//
// Limits:
//   - limit: max log lines returned (default 100, max 1000). Loki hard cap is also 5000.
//   - hours: lookback window (default 1, max 168 = 7 days). Longer windows are slower.
//   - Query length: max 5,000 chars (our safety cap, not a Loki limit).
func (tp *TelemetryProvider) QueryLogs(ctx context.Context, query string, limit int, hours int) (interface{}, error) {
	if query == "" {
		telemetry.Error("query logs called with empty query")
		return nil, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if hours <= 0 {
		hours = 1
	}
	if hours > 168 {
		hours = 168
	}

	now := time.Now()
	start := now.Add(-time.Duration(hours) * time.Hour)

	endpoint := fmt.Sprintf("%s/loki/api/v1/query_range", tp.lokiURL)
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Add("end", strconv.FormatInt(now.UnixNano(), 10))
	params.Add("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		telemetry.Error("failed to create loki request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	telemetry.Info("executing LogQL query", "query", query[:min(len(query), 100)], "limit", limit, "hours", hours)
	resp, err := tp.httpClient.Do(req)
	if err != nil {
		telemetry.Error("failed to query Loki", "error", err)
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		telemetry.Error("Loki returned non-OK status", "status", resp.StatusCode)
		return nil, fmt.Errorf("Loki returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := parseJSONResponse(resp, &result); err != nil {
		telemetry.Error("failed to parse Loki response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	telemetry.Info("logs query executed successfully")
	return result, nil
}

// QueryTraces fetches a trace by ID from Tempo, or searches using a TraceQL query.
//
// Limits:
//   - If traceID is set: fetches full trace summarized as compact span list. traceID must be valid hex.
//   - If traceID is empty: searches via /api/search using TraceQL (e.g. {resource.service.name="proxy"}).
//   - hours: lookback window for search (default 1, max 168).
//   - limit: max traces returned in search mode (default 20, max 100).
func (tp *TelemetryProvider) QueryTraces(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error) {
	if traceID != "" {
		endpoint := fmt.Sprintf("%s/api/traces/%s", tp.tempoURL, traceID)
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		telemetry.Info("retrieving trace from Tempo", "trace_id", traceID)
		resp, err := tp.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to query Tempo: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			telemetry.Warn("trace not found in Tempo", "trace_id", traceID)
			return nil, fmt.Errorf("trace not found: %s", traceID)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Tempo returned status %d", resp.StatusCode)
		}

		var raw map[string]interface{}
		if err := parseJSONResponse(resp, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		telemetry.Info("trace retrieved successfully", "trace_id", traceID)
		return raw, nil
	}

	// Search mode via TraceQL
	if hours <= 0 {
		hours = 1
	}
	if hours > 168 {
		hours = 168
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	now := time.Now()
	start := now.Add(-time.Duration(hours) * time.Hour)

	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("start", strconv.FormatInt(start.Unix(), 10))
	params.Add("end", strconv.FormatInt(now.Unix(), 10))
	if query != "" {
		params.Add("q", query)
	}

	endpoint := fmt.Sprintf("%s/api/search?%s", tp.tempoURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	telemetry.Info("searching traces in Tempo", "query", query, "hours", hours, "limit", limit)
	resp, err := tp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search Tempo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tempo returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := parseJSONResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	telemetry.Info("trace search completed", "query", query, "hours", hours)
	return result, nil
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
