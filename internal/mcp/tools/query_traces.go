package tools

import (
	"context"
	"fmt"

	"observability-hub/internal/telemetry"
)

// QueryTracesInput represents the input for query_traces tool.
type QueryTracesInput struct {
	TraceID string `json:"trace_id"`
}

// QueryTracesHandler retrieves distributed traces and validates input safety.
type QueryTracesHandler struct {
	queryFunc func(ctx context.Context, traceID string) (interface{}, error)
}

// NewQueryTracesHandler creates a new query traces handler.
func NewQueryTracesHandler(queryFunc func(ctx context.Context, traceID string) (interface{}, error)) *QueryTracesHandler {
	return &QueryTracesHandler{
		queryFunc: queryFunc,
	}
}

// Execute runs the query_traces tool with safety validation.
func (h *QueryTracesHandler) Execute(ctx context.Context, input QueryTracesInput) (interface{}, error) {
	// Validate input
	if err := h.validateInput(input); err != nil {
		telemetry.Warn("traces query validation failed", "error", err)
		return nil, err
	}

	// Execute query through provider
	result, err := h.queryFunc(ctx, input.TraceID)
	if err != nil {
		telemetry.Error("traces query execution failed", "error", err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	telemetry.Info("traces query handler executed successfully")
	return result, nil
}

// validateInput performs safety checks on trace ID.
func (h *QueryTracesHandler) validateInput(input QueryTracesInput) error {
	traceID := input.TraceID

	if traceID == "" {
		return fmt.Errorf("trace_id cannot be empty")
	}

	// Trace IDs are typically hex strings, 16 or 32 characters
	if len(traceID) > 128 {
		telemetry.Warn("trace_id exceeds max length", "trace_id_len", len(traceID))
		return fmt.Errorf("trace_id too long (max 128 chars)")
	}

	// Basic validation: trace IDs should be alphanumeric (hex)
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
