package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func (s *KubernetesService) Metrics(ctx context.Context, opts MetricsOptions) ([]Metric, error) {
	if s.signals == nil || s.signals.prometheusURL == "" {
		return nil, fmt.Errorf("PROMETHEUS_URL is required for service metrics")
	}
	if opts.Window <= 0 {
		opts.Window = 5 * time.Minute
	}

	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	metrics := make([]Metric, 0)
	for _, detail := range details {
		pods, err := s.podsFor(ctx, detail)
		if err != nil {
			return nil, err
		}
		if len(pods) == 0 {
			continue
		}

		podMatcher := podRegex(pods)
		queries := []struct {
			signal string
			query  string
		}{
			{
				signal: "cpu_cores",
				query:  fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace=%q,pod=~%q,container!="",container!="POD"}[%s]))`, detail.Namespace, podMatcher, promDuration(opts.Window)),
			},
			{
				signal: "memory_bytes",
				query:  fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace=%q,pod=~%q,container!="",container!="POD"})`, detail.Namespace, podMatcher),
			},
			{
				signal: "restarts",
				query:  fmt.Sprintf(`sum(kube_pod_container_status_restarts_total{namespace=%q,pod=~%q})`, detail.Namespace, podMatcher),
			},
		}

		for _, query := range queries {
			value, err := s.signals.queryMetric(ctx, query.query)
			if err != nil {
				return nil, err
			}
			metrics = append(metrics, Metric{
				Name:      detail.Name,
				Namespace: detail.Namespace,
				Kind:      detail.Kind,
				Signal:    query.signal,
				Value:     value,
			})
		}
	}

	sortMetrics(metrics)
	return metrics, nil
}

func (s *KubernetesService) Traces(ctx context.Context, opts TracesOptions) ([]Trace, error) {
	if s.signals == nil || s.signals.tempoURL == "" {
		return nil, fmt.Errorf("TEMPO_URL is required for service traces")
	}
	if opts.Hours <= 0 {
		opts.Hours = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	traces := make([]Trace, 0)
	for _, detail := range details {
		query := fmt.Sprintf(`{resource.service.name=%q}`, detail.Name)
		found, err := s.signals.searchTraces(ctx, query, opts.Hours, opts.Limit)
		if err != nil {
			return nil, err
		}
		for _, trace := range found {
			trace.Name = detail.Name
			trace.Namespace = detail.Namespace
			trace.Kind = detail.Kind
			traces = append(traces, trace)
		}
	}

	sortTraces(traces)
	if opts.Limit > 0 && len(traces) > opts.Limit {
		traces = traces[:opts.Limit]
	}
	return traces, nil
}

func podRegex(pods []corev1.Pod) string {
	names := make([]string, 0, len(pods))
	for _, pod := range pods {
		names = append(names, regexp.QuoteMeta(pod.Name))
	}
	sort.Strings(names)
	return strings.Join(names, "|")
}

func promDuration(duration time.Duration) string {
	if duration < time.Second {
		duration = time.Second
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

type signalClient struct {
	prometheusURL string
	tempoURL      string
	httpClient    *http.Client
	now           func() time.Time
}

type promQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type tempoSearchResponse struct {
	Traces []tempoTrace `json:"traces"`
}

type tempoTrace struct {
	TraceID           string `json:"traceID"`
	RootServiceName   string `json:"rootServiceName"`
	StartTimeUnixNano string `json:"startTimeUnixNano"`
	DurationMs        int64  `json:"durationMs"`
}

func newSignalClient(prometheusURL string, tempoURL string, httpClient *http.Client) *signalClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &signalClient{
		prometheusURL: strings.TrimRight(prometheusURL, "/"),
		tempoURL:      strings.TrimRight(tempoURL, "/"),
		httpClient:    httpClient,
		now:           time.Now,
	}
}

func (c *signalClient) queryMetric(ctx context.Context, query string) (string, error) {
	params := url.Values{}
	params.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.prometheusURL+"/api/v1/query?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("create metrics request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("query metrics: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return "", fmt.Errorf("query metrics: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded promQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode metrics response: %w", err)
	}
	if decoded.Status != "success" {
		return "", fmt.Errorf("query metrics: prometheus status %q", decoded.Status)
	}
	if len(decoded.Data.Result) == 0 || len(decoded.Data.Result[0].Value) < 2 {
		return "n/a", nil
	}

	value, ok := decoded.Data.Result[0].Value[1].(string)
	if !ok || value == "" {
		return "n/a", nil
	}
	return value, nil
}

func (c *signalClient) searchTraces(ctx context.Context, query string, hours int, limit int) ([]Trace, error) {
	if hours <= 0 {
		hours = 1
	}
	if hours > 168 {
		hours = 168
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	now := c.now()
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("start", strconv.FormatInt(now.Add(-time.Duration(hours)*time.Hour).Unix(), 10))
	params.Set("end", strconv.FormatInt(now.Unix(), 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.tempoURL+"/api/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create traces request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query traces: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("query traces: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded tempoSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode traces response: %w", err)
	}

	traces := make([]Trace, 0, len(decoded.Traces))
	for _, item := range decoded.Traces {
		traces = append(traces, Trace{
			TraceID:         item.TraceID,
			RootServiceName: item.RootServiceName,
			StartTime:       formatUnixNano(item.StartTimeUnixNano),
			Duration:        formatDurationMs(item.DurationMs),
		})
	}
	return traces, nil
}

func formatUnixNano(value string) string {
	if value == "" {
		return "unknown"
	}
	nanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value
	}
	return time.Unix(0, nanos).UTC().Format(time.RFC3339)
}

func formatDurationMs(value int64) string {
	if value <= 0 {
		return "unknown"
	}
	return (time.Duration(value) * time.Millisecond).String()
}
