package tools

import (
	"context"
	"testing"
)

func TestQueryLogsHandler_Execute(t *testing.T) {
	mockQuery := func(ctx context.Context, query string) (interface{}, error) {
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
			query:   string(make([]byte, 10001)),
			wantErr: true,
			errMsg:  "query too long",
		},
		{
			name:    "dangerous keyword delete",
			query:   "DELETE FROM logs",
			wantErr: true,
			errMsg:  "dangerous keyword: delete",
		},
		{
			name:    "dangerous keyword drop",
			query:   "drop table logs",
			wantErr: true,
			errMsg:  "dangerous keyword: drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewQueryLogsHandler(mockQuery)
			_, err := handler.Execute(context.Background(), QueryLogsInput{Query: tt.query})

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
		})
	}
}
