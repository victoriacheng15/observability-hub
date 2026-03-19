package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"observability-hub/internal/telemetry"
)

func TestInstrumentHandler(t *testing.T) {
	// Initialize telemetry in test mode (silences actual exports)
	telemetry.Init(context.Background(), "test-mcp")

	tests := []struct {
		name        string
		toolName    string
		serviceName string
		handler     mcp.ToolHandlerFor[struct{}, string]
		wantErr     bool
	}{
		{
			name:        "Successful tool call",
			toolName:    "test_tool",
			serviceName: "mcp.test",
			handler: func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, string, error) {
				return &mcp.CallToolResult{}, "ok", nil
			},
			wantErr: false,
		},
		{
			name:        "Failed tool call",
			toolName:    "error_tool",
			serviceName: "mcp.test",
			handler: func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, string, error) {
				return nil, "", errors.New("execution failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrumented := InstrumentHandler(tt.toolName, tt.serviceName, tt.handler)

			_, out, err := instrumented(context.Background(), &mcp.CallToolRequest{}, struct{}{})

			if (err != nil) != tt.wantErr {
				t.Errorf("InstrumentHandler() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && out != "ok" {
				t.Errorf("expected output 'ok', got %v", out)
			}
		})
	}
}
