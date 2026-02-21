package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
