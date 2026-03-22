package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"observability-hub/internal/telemetry"
)

var (
	once             sync.Once
	toolCallsCounter telemetry.Int64Counter
	toolDuration     telemetry.Int64Histogram
)

func initTelemetry() {
	once.Do(func() {
		meter := telemetry.GetMeter("mcp")
		var err error
		toolCallsCounter, err = telemetry.NewInt64Counter(meter, "mcp_tool_calls_total", "Total number of MCP tool calls")
		if err != nil {
			telemetry.Error("failed to create mcp_tool_calls_total metric", "error", err)
		}
		toolDuration, err = telemetry.NewInt64Histogram(meter, "mcp_tool_duration", "Duration of MCP tool calls in milliseconds", "ms")
		if err != nil {
			telemetry.Error("failed to create mcp_tool_duration metric", "error", err)
		}
	})
}

// InstrumentHandler wraps an MCP tool handler with tracing and metrics.
func InstrumentHandler[I any, O any](name string, service string, handler mcp.ToolHandlerFor[I, O]) mcp.ToolHandlerFor[I, O] {
	initTelemetry()

	return func(ctx context.Context, req *mcp.CallToolRequest, input I) (*mcp.CallToolResult, O, error) {
		tracer := telemetry.GetTracer("mcp")
		ctx, span := tracer.Start(ctx, fmt.Sprintf("mcp.tool.%s", name))
		defer span.End()

		span.SetAttributes(
			telemetry.StringAttribute("mcp.tool", name),
			telemetry.StringAttribute("mcp.service", service),
		)

		start := time.Now()
		res, out, err := handler(ctx, req, input)
		duration := time.Since(start)

		status := "success"
		if err != nil {
			status = "error"
			span.SetStatus(telemetry.CodeError, err.Error())
			span.RecordError(err)
		}

		if toolCallsCounter != nil {
			telemetry.AddInt64Counter(ctx, toolCallsCounter, 1,
				telemetry.StringAttribute("tool", name),
				telemetry.StringAttribute("service", service),
				telemetry.StringAttribute("status", status),
			)
		}

		if toolDuration != nil {
			telemetry.RecordInt64Histogram(ctx, toolDuration, duration.Milliseconds(),
				telemetry.StringAttribute("tool", name),
				telemetry.StringAttribute("service", service),
			)
		}

		return res, out, err
	}
}
