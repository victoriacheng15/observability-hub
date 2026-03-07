package tools

import (
	"context"
	"strings"
	"testing"
)

func TestQueryTracesHandler_Execute(t *testing.T) {
	mockQuery := func(ctx context.Context, traceID string) (interface{}, error) {
		return map[string]interface{}{"spans": []interface{}{}}, nil
	}

	tests := []struct {
		name    string
		traceID string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid 32-char hex trace ID",
			traceID: "4bf92f3577b34da6a3ce929d0e0e4736",
			wantErr: false,
		},
		{
			name:    "valid uppercase hex trace ID",
			traceID: "4BF92F3577B34DA6A3CE929D0E0E4736",
			wantErr: false,
		},
		{
			name:    "valid 16-char hex trace ID",
			traceID: "4bf92f3577b34da6",
			wantErr: false,
		},
		{
			name:    "empty trace ID",
			traceID: "",
			wantErr: true,
			errMsg:  "trace_id cannot be empty",
		},
		{
			name:    "trace ID too long",
			traceID: string(make([]byte, 129)),
			wantErr: true,
			errMsg:  "trace_id too long",
		},
		{
			name:    "invalid hex character 'z'",
			traceID: "4bf92f3577b34da6a3ce929d0e0e4736zz",
			wantErr: true,
			errMsg:  "must be hexadecimal",
		},
		{
			name:    "invalid non-hex string",
			traceID: "not-a-trace-id",
			wantErr: true,
			errMsg:  "must be hexadecimal",
		},
		{
			name:    "hyphen separator (invalid)",
			traceID: "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
			wantErr: true,
			errMsg:  "must be hexadecimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewQueryTracesHandler(mockQuery)
			_, err := handler.Execute(context.Background(), QueryTracesInput{TraceID: tt.traceID})

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
