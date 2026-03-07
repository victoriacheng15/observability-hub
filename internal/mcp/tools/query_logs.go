package tools

import (
	"context"
	"fmt"
	"strings"

	"observability-hub/internal/telemetry"
)

// QueryLogsInput represents the input for query_logs tool.
type QueryLogsInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"` // max log lines to return, default 100
	Hours int    `json:"hours,omitempty"` // how many hours to look back, default 1, max 168 (7 days)
}

// QueryLogsHandler executes a LogQL query and validates input safety.
type QueryLogsHandler struct {
	queryFunc func(ctx context.Context, query string, limit int, hours int) (interface{}, error)
}

// NewQueryLogsHandler creates a new query logs handler.
func NewQueryLogsHandler(queryFunc func(ctx context.Context, query string, limit int, hours int) (interface{}, error)) *QueryLogsHandler {
	return &QueryLogsHandler{
		queryFunc: queryFunc,
	}
}

// Execute runs the query_logs tool with safety validation.
func (h *QueryLogsHandler) Execute(ctx context.Context, input QueryLogsInput) (interface{}, error) {
	if err := h.validateInput(input); err != nil {
		telemetry.Warn("logs query validation failed", "error", err)
		return nil, err
	}

	result, err := h.queryFunc(ctx, input.Query, input.Limit, input.Hours)
	if err != nil {
		telemetry.Error("logs query execution failed", "error", err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	telemetry.Info("logs query handler executed successfully")
	return result, nil
}

// validateInput performs safety checks on LogQL query.
func (h *QueryLogsHandler) validateInput(input QueryLogsInput) error {
	query := input.Query

	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	if len(query) > 5000 {
		telemetry.Warn("logs query exceeds max length", "query_len", len(query))
		return fmt.Errorf("query too long (max 5000 chars)")
	}

	lower := strings.ToLower(query)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			telemetry.Warn("dangerous keyword detected in logs query", "keyword", pattern)
			return fmt.Errorf("query contains potentially dangerous keyword: %s", pattern)
		}
	}

	return nil
}
