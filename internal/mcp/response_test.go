package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewJSONToolHandler(t *testing.T) {
	handler := NewJSONToolHandler("test_json", "svc", func(ctx context.Context, input struct {
		Value string `json:"value"`
	}) (interface{}, error) {
		return map[string]string{"value": input.Value}, nil
	})

	res, _, err := handler(context.Background(), nil, struct {
		Value string `json:"value"`
	}{Value: "ok"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	text := res.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, `"value":"ok"`) {
		t.Fatalf("got %q, want JSON response", text)
	}
}

func TestNewJSONToolHandler_ReturnsMarshalError(t *testing.T) {
	handler := NewJSONToolHandler("test_json", "svc", func(ctx context.Context, input struct{}) (interface{}, error) {
		return map[string]interface{}{"bad": make(chan int)}, nil
	})

	_, _, err := handler(context.Background(), nil, struct{}{})
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshal MCP tool response") {
		t.Fatalf("got %v, want marshal context", err)
	}
}

func TestNewTextToolHandler(t *testing.T) {
	handler := NewTextToolHandler("test_text", "svc", func(ctx context.Context, input string) (string, error) {
		return "plain text", nil
	})

	res, _, err := handler(context.Background(), nil, "input")
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	text := res.Content[0].(*sdkmcp.TextContent).Text
	if text != "plain text" {
		t.Fatalf("got %q, want plain text", text)
	}
}

func TestNewTextToolHandler_PropagatesExecuteError(t *testing.T) {
	wantErr := errors.New("execute failed")
	handler := NewTextToolHandler("test_text", "svc", func(ctx context.Context, input string) (string, error) {
		return "", wantErr
	})

	_, _, err := handler(context.Background(), nil, "input")
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}
