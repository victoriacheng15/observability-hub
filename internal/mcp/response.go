package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func jsonResult(v any) (*mcp.CallToolResult, error) {
	text, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal MCP tool response: %w", err)
	}
	return textResult(string(text)), nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func NewJSONToolHandler[I any](name, serviceName string, execute func(context.Context, I) (interface{}, error)) mcp.ToolHandlerFor[I, any] {
	return InstrumentHandler(name, serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input I) (*mcp.CallToolResult, any, error) {
		result, err := execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		res, err := jsonResult(result)
		if err != nil {
			return nil, nil, err
		}
		return res, nil, nil
	})
}

func NewTextToolHandler[I any](name, serviceName string, execute func(context.Context, I) (string, error)) mcp.ToolHandlerFor[I, any] {
	return InstrumentHandler(name, serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input I) (*mcp.CallToolResult, any, error) {
		result, err := execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		return textResult(result), nil, nil
	})
}
