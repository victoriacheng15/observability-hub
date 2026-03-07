package tools

import (
	"context"
	"fmt"
	"strings"

	"observability-hub/internal/telemetry"
)

// dangerousPatterns are keywords that must not appear in any query.
var dangerousPatterns = []string{
	"delete",
	"drop",
	"truncate",
	"insert",
	"update",
	"create",
	"alter",
}

// QueryMetricsInput represents the input for query_metrics tool.
type QueryMetricsInput struct {
	Query string `json:"query"`
}

// QueryMetricsHandler executes a PromQL query and validates input safety.
type QueryMetricsHandler struct {
	queryFunc func(ctx context.Context, query string) (interface{}, error)
}

// NewQueryMetricsHandler creates a new query metrics handler.
func NewQueryMetricsHandler(queryFunc func(ctx context.Context, query string) (interface{}, error)) *QueryMetricsHandler {
	return &QueryMetricsHandler{
		queryFunc: queryFunc,
	}
}

// Execute runs the query_metrics tool with safety validation.
func (h *QueryMetricsHandler) Execute(ctx context.Context, input QueryMetricsInput) (interface{}, error) {
	// Validate input
	if err := h.validateInput(input); err != nil {
		telemetry.Warn("query validation failed", "error", err)
		return nil, err
	}

	// Execute query through provider
	result, err := h.queryFunc(ctx, input.Query)
	if err != nil {
		telemetry.Error("query execution failed", "error", err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	telemetry.Info("query handler executed successfully")
	return result, nil
}

// validateInput performs safety checks on PromQL query.
func (h *QueryMetricsHandler) validateInput(input QueryMetricsInput) error {
	query := input.Query

	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	if len(query) > 5000 {
		telemetry.Warn("query exceeds max length", "query_len", len(query))
		return fmt.Errorf("query too long (max 5000 chars)")
	}

	lower := strings.ToLower(query)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			telemetry.Warn("dangerous keyword detected in query", "keyword", pattern)
			return fmt.Errorf("query contains potentially dangerous keyword: %s", pattern)
		}
	}

	return nil
}
