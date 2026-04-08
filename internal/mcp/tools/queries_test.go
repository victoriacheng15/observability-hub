package tools

import (
	"context"
	"strings"
	"testing"
)

func TestQueryMetricsHandler_Execute(t *testing.T) {
	mockQuery := func(ctx context.Context, query string) (interface{}, error) {
		return map[string]interface{}{"status": "success"}, nil
	}

	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple query",
			query:   "up",
			wantErr: false,
		},
		{
			name:    "valid complex query",
			query:   "rate(http_requests_total[5m]) > 0.5",
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
			errMsg:  "query cannot be empty",
		},
		{
			name:    "query too long",
			query:   string(make([]byte, 5001)),
			wantErr: true,
			errMsg:  "query too long",
		},
		{
			name:    "dangerous keyword delete",
			query:   "DELETE FROM metrics",
			wantErr: true,
			errMsg:  "potentially dangerous keyword: delete",
		},
		{
			name:    "dangerous keyword drop",
			query:   "DROP TABLE metrics",
			wantErr: true,
			errMsg:  "potentially dangerous keyword: drop",
		},
		{
			name:    "dangerous keyword insert",
			query:   "INSERT INTO metrics VALUES (1)",
			wantErr: true,
			errMsg:  "potentially dangerous keyword: insert",
		},
		{
			name:    "case insensitive delete",
			query:   "dElEtE",
			wantErr: true,
			errMsg:  "potentially dangerous keyword: delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewQueryMetricsHandler(mockQuery)
			// Point to a non-existent binary to test fail-open logic
			handler.processorPath = "/tmp/non-existent-binary"

			result, err := handler.Execute(context.Background(), QueryMetricsInput{Query: tt.query})

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("got error %q, want error containing %q", err.Error(), tt.errMsg)
				}
			}

			// For valid queries, ensure we get raw metrics back (fail-open)
			if !tt.wantErr && err == nil {
				if result == nil {
					t.Error("expected non-nil result for fail-open scenario")
				}
			}
		})
	}
}

func TestQueryLogsHandler_Execute(t *testing.T) {
	mockQuery := func(ctx context.Context, query string, limit int, hours int) (interface{}, error) {
		return map[string]interface{}{"streams": []interface{}{}}, nil
	}

	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid label matcher",
			query:   `{job="prometheus"}`,
			wantErr: false,
		},
		{
			name:    "valid complex query",
			query:   `{level="error"} | json | message != ""`,
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
			errMsg:  "query cannot be empty",
		},
		{
			name:    "query too long",
			query:   string(make([]byte, 5001)),
			wantErr: true,
			errMsg:  "query too long",
		},
		{
			name:    "dangerous keyword delete",
			query:   "DELETE FROM logs",
			wantErr: true,
			errMsg:  "potentially dangerous keyword: delete",
		},
		{
			name:    "dangerous keyword drop",
			query:   "drop table logs",
			wantErr: true,
			errMsg:  "potentially dangerous keyword: drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewQueryLogsHandler(mockQuery)
			// Point to a non-existent binary to test fail-open logic
			handler.logProcessorPath = "/tmp/non-existent-binary"

			result, err := handler.Execute(context.Background(), QueryLogsInput{Query: tt.query})

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("got error %q, want error containing %q", err.Error(), tt.errMsg)
				}
			}

			// For valid queries, ensure we get raw logs back (fail-open)
			if !tt.wantErr && err == nil {
				if result == nil {
					t.Error("expected non-nil result for fail-open scenario")
				}
			}
		})
	}
}

func TestQueryTracesHandler_Execute(t *testing.T) {
	mockQuery := func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error) {
		return map[string]interface{}{"traces": []interface{}{}}, nil
	}

	tests := []struct {
		name    string
		input   QueryTracesInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid 32-char hex trace ID",
			input:   QueryTracesInput{TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"},
			wantErr: false,
		},
		{
			name:    "valid uppercase hex trace ID",
			input:   QueryTracesInput{TraceID: "4BF92F3577B34DA6A3CE929D0E0E4736"},
			wantErr: false,
		},
		{
			name:    "valid 16-char hex trace ID",
			input:   QueryTracesInput{TraceID: "4bf92f3577b34da6"},
			wantErr: false,
		},
		{
			name:    "search with traceql query",
			input:   QueryTracesInput{Query: `{resource.service.name="analytics"}`, Limit: 10},
			wantErr: false,
		},
		{
			name:    "search with no filter returns all",
			input:   QueryTracesInput{},
			wantErr: false,
		},
		{
			name:    "trace ID too long",
			input:   QueryTracesInput{TraceID: string(make([]byte, 129))},
			wantErr: true,
			errMsg:  "trace_id too long",
		},
		{
			name:    "invalid hex character",
			input:   QueryTracesInput{TraceID: "4bf92f3577b34da6zz"},
			wantErr: true,
			errMsg:  "must be hexadecimal",
		},
		{
			name:    "hyphen separator (invalid)",
			input:   QueryTracesInput{TraceID: "4bf92f35-77b3-4da6-a3ce-929d0e0e4736"},
			wantErr: true,
			errMsg:  "must be hexadecimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewQueryTracesHandler(mockQuery)
			_, err := handler.Execute(context.Background(), tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("got error %q, want error containing %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
