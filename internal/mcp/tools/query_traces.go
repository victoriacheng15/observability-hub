package tools

import (
	"context"
	"fmt"
	"strconv"

	"observability-hub/internal/telemetry"
)

// QueryTracesInput represents the input for query_traces tool.
// Provide trace_id to fetch a specific trace, or query (TraceQL) to search.
type QueryTracesInput struct {
	TraceID string `json:"trace_id,omitempty"` // hex trace ID — fetches full trace when set
	Query   string `json:"query,omitempty"`    // TraceQL query e.g. {resource.service.name="collectors"}
	Hours   int    `json:"hours,omitempty"`    // lookback window in hours for search mode (default 1, max 168)
	Limit   int    `json:"limit,omitempty"`    // max results in search mode (default 20)
}

// QueryTracesHandler retrieves distributed traces and validates input safety.
type QueryTracesHandler struct {
	queryFunc func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error)
}

// NewQueryTracesHandler creates a new query traces handler.
func NewQueryTracesHandler(queryFunc func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error)) *QueryTracesHandler {
	return &QueryTracesHandler{queryFunc: queryFunc}
}

// Execute runs the query_traces tool. Validates trace_id if set, then delegates to provider.
// Fetch-by-ID responses are summarized into a compact, AI-friendly format.
func (h *QueryTracesHandler) Execute(ctx context.Context, input QueryTracesInput) (interface{}, error) {
	if input.TraceID != "" {
		if err := validateTraceID(input.TraceID); err != nil {
			telemetry.Warn("trace ID validation failed", "error", err)
			return nil, err
		}
	}

	raw, err := h.queryFunc(ctx, input.TraceID, input.Query, input.Hours, input.Limit)
	if err != nil {
		telemetry.Error("traces query execution failed", "error", err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	// Summarize full trace responses into compact format for AI consumption
	if input.TraceID != "" {
		if rawMap, ok := raw.(map[string]interface{}); ok {
			telemetry.Info("traces query handler executed successfully")
			return summarizeTrace(input.TraceID, rawMap), nil
		}
	}

	telemetry.Info("traces query handler executed successfully")
	return raw, nil
}

// validateTraceID checks that the trace ID is a valid hex string.
func validateTraceID(traceID string) error {
	if len(traceID) > 128 {
		telemetry.Warn("trace_id exceeds max length", "trace_id_len", len(traceID))
		return fmt.Errorf("trace_id too long (max 128 chars)")
	}
	for _, ch := range traceID {
		if !isHexChar(ch) {
			telemetry.Warn("invalid trace_id format", "char", string(ch))
			return fmt.Errorf("trace_id must be hexadecimal")
		}
	}
	return nil
}

// isHexChar checks if a rune is a valid hex character.
func isHexChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// summarizeTrace converts a raw OTLP trace payload into a compact, AI-friendly summary.
func summarizeTrace(traceID string, raw map[string]interface{}) map[string]interface{} {
	type spanSummary struct {
		Service    string  `json:"service"`
		Scope      string  `json:"scope"`
		Name       string  `json:"name"`
		DurationMs float64 `json:"duration_ms"`
		Error      bool    `json:"error,omitempty"`
	}

	var spans []spanSummary
	var minStart, maxEnd int64

	for _, b := range toList(raw["batches"]) {
		batch, _ := b.(map[string]interface{})
		svc := "unknown"
		if res, ok := batch["resource"].(map[string]interface{}); ok {
			for _, a := range toList(res["attributes"]) {
				attr, _ := a.(map[string]interface{})
				if attrStr(attr["key"]) == "service.name" {
					if val, ok := attr["value"].(map[string]interface{}); ok {
						svc = attrStr(val["stringValue"])
					}
				}
			}
		}
		for _, ss := range toList(batch["scopeSpans"]) {
			ss_, _ := ss.(map[string]interface{})
			scope := ""
			if sc, ok := ss_["scope"].(map[string]interface{}); ok {
				scope = attrStr(sc["name"])
			}
			for _, s := range toList(ss_["spans"]) {
				sp, _ := s.(map[string]interface{})
				start := parseNano(sp["startTimeUnixNano"])
				end := parseNano(sp["endTimeUnixNano"])
				isErr := false
				if status, ok := sp["status"].(map[string]interface{}); ok {
					isErr = attrStr(status["code"]) == "STATUS_CODE_ERROR"
				}
				spans = append(spans, spanSummary{
					Service:    svc,
					Scope:      scope,
					Name:       attrStr(sp["name"]),
					DurationMs: float64(end-start) / 1e6,
					Error:      isErr,
				})
				if minStart == 0 || start < minStart {
					minStart = start
				}
				if end > maxEnd {
					maxEnd = end
				}
			}
		}
	}

	return map[string]interface{}{
		"trace_id":          traceID,
		"span_count":        len(spans),
		"total_duration_ms": float64(maxEnd-minStart) / 1e6,
		"spans":             spans,
	}
}

func toList(v interface{}) []interface{} {
	if l, ok := v.([]interface{}); ok {
		return l
	}
	return nil
}

func attrStr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseNano(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}
