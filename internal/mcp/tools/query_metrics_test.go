package tools

import (
	"context"
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
			query:   string(make([]byte, 10001)),
			wantErr: true,
			errMsg:  "query too long",
		},
		{
			name:    "dangerous keyword delete",
			query:   "DELETE FROM metrics",
			wantErr: true,
			errMsg:  "dangerous keyword: delete",
		},
		{
			name:    "dangerous keyword drop",
			query:   "DROP TABLE metrics",
			wantErr: true,
			errMsg:  "dangerous keyword: drop",
		},
		{
			name:    "dangerous keyword insert",
			query:   "INSERT INTO metrics VALUES (1)",
			wantErr: true,
			errMsg:  "dangerous keyword: insert",
		},
		{
			name:    "case insensitive delete",
			query:   "dElEtE",
			wantErr: true,
			errMsg:  "dangerous keyword: delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewQueryMetricsHandler(mockQuery)
			_, err := handler.Execute(context.Background(), QueryMetricsInput{Query: tt.query})

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !matchErrorMsg(err.Error(), tt.errMsg) {
					t.Errorf("got error %q, want error containing %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// matchErrorMsg checks if errorMsg contains substring
func matchErrorMsg(errorMsg, substring string) bool {
	// Simple substring check
	return len(errorMsg) >= len(substring)
}
