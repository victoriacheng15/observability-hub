package tools

import (
	"context"
	"strings"
	"testing"
)

func TestInvestigateIncidentHandler_Execute(t *testing.T) {
	noopMetrics := func(ctx context.Context, query string) (interface{}, error) {
		return map[string]interface{}{"data": map[string]interface{}{"result": []interface{}{}}}, nil
	}
	noopLogs := func(ctx context.Context, query string, limit int, hours int) (interface{}, error) {
		return map[string]interface{}{"data": map[string]interface{}{"result": []interface{}{}}}, nil
	}
	noopTraces := func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error) {
		return map[string]interface{}{"traces": []interface{}{}}, nil
	}

	logsWithErrors := func(ctx context.Context, query string, limit int, hours int) (interface{}, error) {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"result": []interface{}{"error: connection refused"},
			},
		}, nil
	}
	tracesWithErrors := func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error) {
		return map[string]interface{}{
			"traces": []interface{}{
				map[string]interface{}{"traceID": "abc123", "rootServiceName": "proxy"},
			},
		}, nil
	}

	tests := []struct {
		name        string
		input       InvestigateIncidentInput
		mockMetrics func(context.Context, string) (interface{}, error)
		mockLogs    func(context.Context, string, int, int) (interface{}, error)
		mockTraces  func(context.Context, string, string, int, int) (interface{}, error)
		wantErr     bool
		errMsg      string
		wantHealthy bool
		wantSummary string
	}{
		{
			name:        "healthy service returns no errors",
			input:       InvestigateIncidentInput{Service: "proxy", Hours: 1},
			mockMetrics: noopMetrics,
			mockLogs:    noopLogs,
			mockTraces:  noopTraces,
			wantHealthy: true,
		},
		{
			name:        "error logs detected marks unhealthy",
			input:       InvestigateIncidentInput{Service: "proxy", Hours: 1},
			mockMetrics: noopMetrics,
			mockLogs:    logsWithErrors,
			mockTraces:  noopTraces,
			wantHealthy: false,
			wantSummary: "Error log entries found",
		},
		{
			name:        "error traces detected marks unhealthy",
			input:       InvestigateIncidentInput{Service: "proxy", Hours: 1},
			mockMetrics: noopMetrics,
			mockLogs:    noopLogs,
			mockTraces:  tracesWithErrors,
			wantHealthy: false,
			wantSummary: "Error spans found",
		},
		{
			name:        "both logs and traces with errors",
			input:       InvestigateIncidentInput{Service: "collectors", Hours: 6},
			mockMetrics: noopMetrics,
			mockLogs:    logsWithErrors,
			mockTraces:  tracesWithErrors,
			wantHealthy: false,
			wantSummary: "Error log entries found",
		},
		{
			name:    "missing service returns error",
			input:   InvestigateIncidentInput{},
			wantErr: true,
			errMsg:  "service is required",
		},
		{
			name:    "invalid since format returns error",
			input:   InvestigateIncidentInput{Service: "proxy", Since: "not-a-date"},
			wantErr: true,
			errMsg:  "invalid since format",
		},
		{
			name:        "since in the past overrides hours",
			input:       InvestigateIncidentInput{Service: "proxy", Since: "2006-01-02T15:04:05Z"},
			mockMetrics: noopMetrics,
			mockLogs:    noopLogs,
			mockTraces:  noopTraces,
			wantHealthy: true,
		},
		{
			name:    "since in the future returns error",
			input:   InvestigateIncidentInput{Service: "proxy", Since: "2099-01-01T00:00:00Z"},
			wantErr: true,
			errMsg:  "since must be in the past",
		},
		{
			name:        "default hours applied when zero",
			input:       InvestigateIncidentInput{Service: "proxy"},
			mockMetrics: noopMetrics,
			mockLogs:    noopLogs,
			mockTraces:  noopTraces,
			wantHealthy: true,
		},
		{
			name:        "hours capped at 168",
			input:       InvestigateIncidentInput{Service: "proxy", Hours: 999},
			mockMetrics: noopMetrics,
			mockLogs:    noopLogs,
			mockTraces:  noopTraces,
			wantHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewInvestigateIncidentHandler(tt.mockMetrics, tt.mockLogs, tt.mockTraces)
			result, err := handler.Execute(context.Background(), tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("got error %q, want error containing %q", err.Error(), tt.errMsg)
			}
			if tt.wantErr {
				return
			}

			report, ok := result.(IncidentReport)
			if !ok {
				t.Fatalf("expected IncidentReport, got %T", result)
			}
			if report.Healthy != tt.wantHealthy {
				t.Errorf("got healthy=%v, want healthy=%v", report.Healthy, tt.wantHealthy)
			}
			if tt.wantSummary != "" && !strings.Contains(report.ErrorSummary, tt.wantSummary) {
				t.Errorf("got summary %q, want it to contain %q", report.ErrorSummary, tt.wantSummary)
			}
		})
	}
}
