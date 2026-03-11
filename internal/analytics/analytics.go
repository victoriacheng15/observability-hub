package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// MetricBatch represents a collection of metric samples keyed by type (cpu, ram, disk, network, temp).
type MetricBatch map[string][]Sample

// Sample represents a single data point with a timestamp and a JSONB-ready payload.
type Sample struct {
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// ThanosClient handles communication with the Thanos Query API for batch metric retrieval.
type ThanosClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewThanosClient creates a new client for querying Thanos.
func NewThanosClient(baseURL string) *ThanosClient {
	return &ThanosClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// QueryRange fetches a range of metrics for a given PromQL query.
func (c *ThanosClient) QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]Sample, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", start.Unix()))
	params.Add("end", fmt.Sprintf("%d", end.Unix()))
	params.Add("step", step)

	reqURL := fmt.Sprintf("%s/api/v1/query_range?%s", c.BaseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("thanos api returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]interface{}   `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Status != "success" {
		return nil, fmt.Errorf("thanos api reported failure status: %s", apiResp.Status)
	}

	var samples []Sample
	for _, res := range apiResp.Data.Result {
		for _, val := range res.Values {
			if len(val) < 2 {
				continue
			}
			tsFloat, ok1 := val[0].(float64)
			valStr, ok2 := val[1].(string)
			if !ok1 || !ok2 {
				continue
			}

			samples = append(samples, Sample{
				Timestamp: time.Unix(int64(tsFloat), 0),
				Payload: map[string]interface{}{
					"value":  valStr,
					"labels": res.Metric,
				},
			})
		}
	}

	return samples, nil
}

// ThanosResourceProvider implements ResourceProvider using Thanos.
type ThanosResourceProvider struct {
	Client *ThanosClient
}

func NewThanosResourceProvider(client *ThanosClient) *ThanosResourceProvider {
	return &ThanosResourceProvider{Client: client}
}

func (p *ThanosResourceProvider) GetEnergyJoules(ctx context.Context, start, end time.Time) (float64, error) {
	// Query for node energy increase over the period
	query := fmt.Sprintf("sum(increase(kepler_node_cpu_joules_total[%s]))", "15m")
	samples, err := p.Client.QueryRange(ctx, query, start, end, "1m")
	if err != nil {
		return 0, err
	}

	if len(samples) == 0 {
		return 0, nil
	}

	// Sum up the values from all samples in the range (though sum() already does most work)
	// We'll take the last value as it's the most complete increase for the period
	last := samples[len(samples)-1]
	val, _ := strconv.ParseFloat(fmt.Sprintf("%v", last.Payload["value"]), 64)
	return val, nil
}

func (p *ThanosResourceProvider) GetContainerEnergy(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	// Query energy increase per container
	query := fmt.Sprintf("sum(increase(kepler_container_cpu_joules_total[%s])) by (container_name)", "15m")
	samples, err := p.Client.QueryRange(ctx, query, start, end, "1m")
	if err != nil {
		return nil, err
	}

	features := make(map[string]float64)
	for _, s := range samples {
		labels, ok := s.Payload["labels"].(map[string]string)
		if !ok {
			continue
		}
		container := labels["container_name"]
		if container == "" || container == "POD" {
			continue
		}

		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		features[container] = val
	}
	return features, nil
}

func (p *ThanosResourceProvider) GetHostServiceCPU(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	// Query CPU usage for target systemd services
	// Pattern: node_systemd_unit_cpu_usage_seconds_total{name=~"proxy.service|ingestion.service|mcp-.*"}
	query := fmt.Sprintf("sum(rate(node_systemd_unit_cpu_usage_seconds_total{name=~\"proxy.service|ingestion.service|mcp-.*\"}[%s])) by (name)", "15m")
	samples, err := p.Client.QueryRange(ctx, query, start, end, "1m")
	if err != nil {
		return nil, err
	}

	serviceCPU := make(map[string]float64)
	for _, s := range samples {
		labels, ok := s.Payload["labels"].(map[string]string)
		if !ok {
			continue
		}
		service := labels["name"]
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		serviceCPU[service] = val
	}
	return serviceCPU, nil
}

func (p *ThanosResourceProvider) GetValueUnits(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	// 1. Define queries for all business value counters
	queries := map[string]string{
		"ingestion": "sum(increase(second_brain_sync_processed_total[15m])) + sum(increase(reading_sync_processed_total[15m]))",
		"proxy":     "sum(increase(proxy_webhook_received_total[15m])) + sum(increase(proxy_synthetic_request_total[15m]))",
	}

	units := make(map[string]float64)
	for feature, query := range queries {
		samples, err := p.Client.QueryRange(ctx, query, start, end, "1m")
		if err != nil {
			continue
		}
		if len(samples) > 0 {
			last := samples[len(samples)-1]
			val, _ := strconv.ParseFloat(fmt.Sprintf("%v", last.Payload["value"]), 64)
			units[feature] = val
		}
	}
	return units, nil
}

func (p *ThanosResourceProvider) GetCarbonIntensity(ctx context.Context) (float64, error) {

	// Default: ~150g CO2 per kWh (Sample value for a "greenish" grid)
	// In a real implementation, this could call an external API.
	return 150.0, nil
}

func (p *ThanosResourceProvider) GetCostFactor(ctx context.Context) (float64, error) {
	// Default: $0.15 CAD per kWh (Sample price)
	// 1 kWh = 3_600_000 Joules
	// Cost per Joule = 0.15 / 3_600_000
	return 0.15 / 3600000.0, nil
}
