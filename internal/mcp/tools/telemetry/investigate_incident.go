package telemetry

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	libtelemetry "observability-hub/internal/telemetry"
)

// InvestigateIncidentInput represents the input for the investigate_incident tool.
type InvestigateIncidentInput struct {
	Service string `json:"service"`         // service name to investigate e.g. "proxy", "analytics"
	Hours   int    `json:"hours,omitempty"` // lookback window in hours (default 1, max 168)
	Since   string `json:"since,omitempty"` // ISO 8601 start time e.g. "2026-03-06T17:00:00Z" — overrides hours
}

// IncidentReport is the structured output of an investigation.
type IncidentReport struct {
	Service  string `json:"service"`
	WindowHr int    `json:"window_hours"`
	Since    string `json:"since,omitempty"`
	Healthy  bool   `json:"healthy"`

	ErrorLogs    interface{} `json:"error_logs,omitempty"`
	ErrorTraces  interface{} `json:"error_traces,omitempty"`
	Metrics      interface{} `json:"metrics,omitempty"`
	ErrorSummary string      `json:"error_summary,omitempty"`
}

// InvestigateIncidentHandler orchestrates metrics, logs, and traces to produce an incident report.
type InvestigateIncidentHandler struct {
	queryMetrics func(ctx context.Context, query string) (interface{}, error)
	queryLogs    func(ctx context.Context, query string, limit int, hours int) (interface{}, error)
	queryTraces  func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error)
}

// NewInvestigateIncidentHandler creates a new investigate_incident handler.
func NewInvestigateIncidentHandler(
	queryMetrics func(ctx context.Context, query string) (interface{}, error),
	queryLogs func(ctx context.Context, query string, limit int, hours int) (interface{}, error),
	queryTraces func(ctx context.Context, traceID string, query string, hours int, limit int) (interface{}, error),
) *InvestigateIncidentHandler {
	return &InvestigateIncidentHandler{
		queryMetrics: queryMetrics,
		queryLogs:    queryLogs,
		queryTraces:  queryTraces,
	}
}

// Execute runs the investigate_incident tool.
// It checks for errors in logs and traces in parallel, then fetches supporting
// metrics if issues are found, and returns a structured incident report.
func (h *InvestigateIncidentHandler) Execute(ctx context.Context, input InvestigateIncidentInput) (interface{}, error) {
	if input.Service == "" {
		return nil, fmt.Errorf("service is required")
	}

	// Resolve hours: since overrides hours when set
	if input.Since != "" {
		t, err := time.Parse(time.RFC3339, input.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid since format, expected RFC3339 e.g. 2026-03-06T17:00:00Z: %w", err)
		}
		computed := int(math.Ceil(time.Since(t).Hours()))
		if computed <= 0 {
			return nil, fmt.Errorf("since must be in the past")
		}
		input.Hours = computed
	}
	if input.Hours <= 0 {
		input.Hours = 1
	}
	if input.Hours > 168 {
		input.Hours = 168
	}

	libtelemetry.Info("investigating incident", "service", input.Service, "hours", input.Hours, "since", input.Since)

	// Step 1: Check for errors in logs and traces in parallel
	type result struct {
		data interface{}
		err  error
	}

	logsCh := make(chan result, 1)
	tracesCh := make(chan result, 1)

	logQuery := fmt.Sprintf(`{service="%s"} |~ "(?i)error"`, input.Service)
	traceQuery := fmt.Sprintf(`{resource.service.name="%s"} && status=error`, input.Service)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		data, err := h.queryLogs(ctx, logQuery, 20, input.Hours)
		logsCh <- result{data, err}
	}()

	go func() {
		defer wg.Done()
		data, err := h.queryTraces(ctx, "", traceQuery, input.Hours, 10)
		tracesCh <- result{data, err}
	}()

	wg.Wait()
	logsResult := <-logsCh
	tracesResult := <-tracesCh

	report := IncidentReport{
		Service:  input.Service,
		WindowHr: input.Hours,
		Since:    input.Since,
		Healthy:  true,
	}

	hasErrors := false

	if logsResult.err == nil && hasLogEntries(logsResult.data) {
		report.ErrorLogs = logsResult.data
		hasErrors = true
	}
	if tracesResult.err == nil && hasTraceEntries(tracesResult.data) {
		report.ErrorTraces = tracesResult.data
		hasErrors = true
	}

	if !hasErrors {
		libtelemetry.Info("incident investigation complete: no errors found", "service", input.Service)
		return report, nil
	}

	// Step 2: Errors found — fetch supporting metrics
	report.Healthy = false
	errorRateQuery := fmt.Sprintf(`sum(rate(http_requests_total{service="%s",status=~"5.."}[5m])) / sum(rate(http_requests_total{service="%s"}[5m]))`, input.Service, input.Service)
	metricsData, err := h.queryMetrics(ctx, errorRateQuery)
	if err == nil {
		report.Metrics = metricsData
	}

	report.ErrorSummary = buildSummary(report)
	libtelemetry.Info("incident investigation complete: errors detected", "service", input.Service)
	return report, nil
}

// hasLogEntries returns true if the log result contains at least one log line.
func hasLogEntries(data interface{}) bool {
	m, ok := data.(map[string]interface{})
	if !ok {
		return false
	}
	results, ok := m["data"].(map[string]interface{})
	if !ok {
		return false
	}
	entries, ok := results["result"].([]interface{})
	return ok && len(entries) > 0
}

// hasTraceEntries returns true if the trace search result contains at least one trace.
func hasTraceEntries(data interface{}) bool {
	m, ok := data.(map[string]interface{})
	if !ok {
		return false
	}
	traces, ok := m["traces"].([]interface{})
	return ok && len(traces) > 0
}

// buildSummary produces a plain-text summary for the AI to reason over.
func buildSummary(r IncidentReport) string {
	summary := fmt.Sprintf("Incident detected for service %q over the last %d hour(s).", r.Service, r.WindowHr)
	if r.ErrorLogs != nil {
		summary += " Error log entries found."
	}
	if r.ErrorTraces != nil {
		summary += " Error spans found in distributed traces."
	}
	if r.Metrics != nil {
		summary += " Error rate metrics retrieved for correlation."
	}
	return summary
}
