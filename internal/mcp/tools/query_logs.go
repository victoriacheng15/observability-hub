package tools

import (
	"context"
	"fmt"

	"observability-hub/internal/telemetry"
)

// QueryLogsInput represents the input for query_logs tool.
type QueryLogsInput struct {
	Query string `json:"query"`
}

// QueryLogsHandler executes a LogQL query and validates input safety.
type QueryLogsHandler struct {
	queryFunc func(ctx context.Context, query string) (interface{}, error)
}

// NewQueryLogsHandler creates a new query logs handler.
func NewQueryLogsHandler(queryFunc func(ctx context.Context, query string) (interface{}, error)) *QueryLogsHandler {
	return &QueryLogsHandler{
		queryFunc: queryFunc,
	}
}

// Execute runs the query_logs tool with safety validation.
func (h *QueryLogsHandler) Execute(ctx context.Context, input QueryLogsInput) (interface{}, error) {
	// Validate input
	if err := h.validateInput(input); err != nil {
		telemetry.Warn("logs query validation failed", "error", err)
		return nil, err
	}

	// Execute query through provider
	result, err := h.queryFunc(ctx, input.Query)
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

	if len(query) > 10000 {
		telemetry.Warn("logs query exceeds max length", "query_len", len(query))
		return fmt.Errorf("query too long (max 10000 chars)")
	}

	// Prevent potentially dangerous patterns
	dangerousPatterns := []string{
		"delete",
		"drop",
		"truncate",
		"insert",
		"update",
		"create",
		"alter",
	}

	for _, pattern := range dangerousPatterns {
		if matchCaseInsensitive(query, pattern) {
			telemetry.Warn("dangerous keyword detected in logs query", "keyword", pattern)
			return fmt.Errorf("query contains potentially dangerous keyword: %s", pattern)
		}
	}

	return nil
}
