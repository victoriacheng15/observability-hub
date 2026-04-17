package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"sort"
	"strings"
)

type LokiResponse struct {
	Status string   `json:"status"`
	Data   LokiData `json:"data"`
}

type LokiData struct {
	ResultType string       `json:"resultType"`
	Result     []LokiStream `json:"result"`
}

type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type MetricResponse struct {
	Status string     `json:"status"`
	Data   MetricData `json:"data"`
}

type MetricData struct {
	ResultType string         `json:"resultType"`
	Result     []MetricResult `json:"result"`
}

type MetricResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value,omitempty"`
	Values [][]any           `json:"values,omitempty"`
}

type LogSummaryResult struct {
	TotalRawLines   int               `json:"total_raw_lines"`
	SummarizedCount int               `json:"summarized_count"`
	Entries         []LogSummaryEntry `json:"entries"`
}

type LogSummaryEntry struct {
	Level            string            `json:"level"`
	Message          string            `json:"message"`
	Count            int               `json:"count"`
	FirstTimestampNS string            `json:"first_timestamp_ns"`
	LastTimestampNS  string            `json:"last_timestamp_ns"`
	Context          map[string]string `json:"context,omitempty"`
}

type logAggregate struct {
	count            int
	firstTimestampNS string
	lastTimestampNS  string
	context          map[string]string
}

type MetricSummaryResult struct {
	ResultType      string               `json:"result_type"`
	TotalRawLines   int                  `json:"total_raw_lines"`
	SummarizedCount int                  `json:"summarized_count"`
	Entries         []MetricSummaryEntry `json:"entries"`
}

type MetricSummaryEntry struct {
	Metric               string            `json:"metric"`
	Kind                 string            `json:"kind"`
	Status               string            `json:"status"`
	Labels               map[string]string `json:"labels"`
	SampleCount          int               `json:"sample_count"`
	Timestamp            *float64          `json:"timestamp,omitempty"`
	Current              *float64          `json:"current,omitempty"`
	Min                  *float64          `json:"min,omitempty"`
	Max                  *float64          `json:"max,omitempty"`
	Avg                  *float64          `json:"avg,omitempty"`
	P95                  *float64          `json:"p95,omitempty"`
	P99                  *float64          `json:"p99,omitempty"`
	First                *float64          `json:"first,omitempty"`
	Last                 *float64          `json:"last,omitempty"`
	TrendDelta           *float64          `json:"trend_delta,omitempty"`
	Delta                *float64          `json:"delta,omitempty"`
	AverageRatePerSecond *float64          `json:"average_rate_per_second,omitempty"`
	ResetsDetected       *int              `json:"resets_detected,omitempty"`
	FirstTimestamp       *float64          `json:"first_timestamp,omitempty"`
	LastTimestamp        *float64          `json:"last_timestamp,omitempty"`
}

func processLokiResponse(response LokiResponse) LogSummaryResult {
	infoEntries := make(map[string]*logAggregate)
	warnEntries := make(map[string]*logAggregate)
	errorEntries := make(map[string]*logAggregate)
	totalLines := 0

	for _, stream := range response.Data.Result {
		level := stream.Stream["level"]
		if level == "" {
			level = stream.Stream["detected_level"]
		}
		if level == "" {
			level = stream.Stream["severity_text"]
		}
		if level == "" {
			level = "info"
		}
		normalized := normalizeLogLevel(strings.ToLower(level))

		for _, entry := range stream.Values {
			totalLines++
			if len(entry) < 2 {
				continue
			}
			timestampNS := entry[0]
			message := entry[1]

			switch normalized {
			case "error":
				recordLogEntry(errorEntries, message, timestampNS, extractLogContext(stream.Stream, "error"))
			case "warn":
				recordLogEntry(warnEntries, message, timestampNS, extractLogContext(stream.Stream, "warn"))
			default:
				recordLogEntry(infoEntries, message, timestampNS, nil)
			}
		}
	}

	finalEntries := make([]LogSummaryEntry, 0)
	appendLogEntries(&finalEntries, "error", errorEntries)
	appendLogEntries(&finalEntries, "warn", warnEntries)
	appendLogEntries(&finalEntries, "info", infoEntries)

	return LogSummaryResult{
		TotalRawLines:   totalLines,
		SummarizedCount: len(finalEntries),
		Entries:         finalEntries,
	}
}

func normalizeLogLevel(level string) string {
	switch level {
	case "error", "err", "fatal", "panic":
		return "error"
	case "warn", "warning":
		return "warn"
	default:
		return "info"
	}
}

func recordLogEntry(entries map[string]*logAggregate, message, timestampNS string, context map[string]string) {
	existing, ok := entries[message]
	if !ok {
		entries[message] = &logAggregate{
			count:            1,
			firstTimestampNS: timestampNS,
			lastTimestampNS:  timestampNS,
			context:          context,
		}
		return
	}

	existing.count++
	if timestampBefore(timestampNS, existing.firstTimestampNS) {
		existing.firstTimestampNS = timestampNS
	}
	if timestampAfter(timestampNS, existing.lastTimestampNS) {
		existing.lastTimestampNS = timestampNS
	}
	mergeLogContext(&existing.context, context)
}

func timestampBefore(left, right string) bool {
	lb, lok := new(big.Int).SetString(left, 10)
	rb, rok := new(big.Int).SetString(right, 10)
	if lok && rok {
		return lb.Cmp(rb) < 0
	}
	return left < right
}

func timestampAfter(left, right string) bool {
	lb, lok := new(big.Int).SetString(left, 10)
	rb, rok := new(big.Int).SetString(right, 10)
	if lok && rok {
		return lb.Cmp(rb) > 0
	}
	return left > right
}

func appendLogEntries(finalEntries *[]LogSummaryEntry, level string, entries map[string]*logAggregate) {
	type pair struct {
		message string
		entry   *logAggregate
	}

	pairs := make([]pair, 0, len(entries))
	for msg, ent := range entries {
		pairs = append(pairs, pair{message: msg, entry: ent})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].entry.count == pairs[j].entry.count {
			return pairs[i].message < pairs[j].message
		}
		return pairs[i].entry.count > pairs[j].entry.count
	})

	for _, p := range pairs {
		*finalEntries = append(*finalEntries, LogSummaryEntry{
			Level:            level,
			Message:          p.message,
			Count:            p.entry.count,
			FirstTimestampNS: p.entry.firstTimestampNS,
			LastTimestampNS:  p.entry.lastTimestampNS,
			Context:          p.entry.context,
		})
	}
}

func extractLogContext(stream map[string]string, scope string) map[string]string {
	var allowed []string
	if scope == "error" {
		allowed = []string{"service_name", "repo", "error", "status", "path", "ref", "action"}
	} else {
		allowed = []string{"service_name", "status", "path", "action", "error"}
	}

	context := make(map[string]string)
	for _, key := range allowed {
		if value, ok := stream[key]; ok && strings.TrimSpace(value) != "" {
			context[key] = trimContextValue(value)
		}
	}

	if scope == "error" {
		if output, ok := stream["output"]; ok {
			if preview := previewOutput(output); preview != "" {
				context["output_preview"] = preview
			}
		}
	}

	if len(context) == 0 {
		return nil
	}
	return context
}

func trimContextValue(value string) string {
	const maxLen = 160
	trimmed := strings.TrimSpace(value)
	runes := []rune(trimmed)
	if len(runes) <= maxLen {
		return trimmed
	}
	return string(runes[:maxLen]) + "..."
}

func previewOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
		if msg, ok := obj["msg"].(string); ok {
			return trimContextValue(msg)
		}
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) > 0 {
		return trimContextValue(lines[0])
	}
	return ""
}

func mergeLogContext(existingContext *map[string]string, newContext map[string]string) {
	if newContext == nil {
		return
	}
	if *existingContext == nil {
		*existingContext = make(map[string]string)
	}
	for key, value := range newContext {
		if existingValue, ok := (*existingContext)[key]; ok {
			if existingValue != value {
				(*existingContext)[key] = "<multiple>"
			}
		} else {
			(*existingContext)[key] = value
		}
	}
}

func processMetricsResponse(response MetricResponse) MetricSummaryResult {
	finalEntries := make([]MetricSummaryEntry, 0)
	seriesCount := len(response.Data.Result)
	resultType := response.Data.ResultType

	for _, series := range response.Data.Result {
		name := series.Metric["__name__"]
		if name == "" {
			name = "unknown"
		}
		labels := metricLabels(series.Metric)

		switch resultType {
		case "vector":
			timestamp, current, ok := parseSample(series.Value)
			if !ok {
				continue
			}
			kind := metricKind(name)
			finalEntries = append(finalEntries, MetricSummaryEntry{
				Metric:      name,
				Kind:        kind,
				Status:      "normal",
				Labels:      labels,
				SampleCount: 1,
				Timestamp:   ptrFloat(timestamp),
				Current:     ptrFloat(current),
			})
		case "matrix":
			samples := make([][2]float64, 0, len(series.Values))
			for _, raw := range series.Values {
				ts, val, ok := parseSample(raw)
				if ok {
					samples = append(samples, [2]float64{ts, val})
				}
			}
			if len(samples) == 0 {
				continue
			}

			firstTimestamp := samples[0][0]
			first := samples[0][1]
			lastTimestamp := samples[len(samples)-1][0]
			last := samples[len(samples)-1][1]
			kind := metricKind(name)

			if kind == "counter" {
				delta, resets := counterDeltaAndResets(samples)
				var avgRate *float64
				elapsed := lastTimestamp - firstTimestamp
				if elapsed > 0 {
					avgRate = ptrFloat(delta / elapsed)
				}
				finalEntries = append(finalEntries, MetricSummaryEntry{
					Metric:               name,
					Kind:                 kind,
					Status:               "normal",
					Labels:               labels,
					SampleCount:          len(samples),
					First:                ptrFloat(first),
					Last:                 ptrFloat(last),
					Delta:                ptrFloat(delta),
					AverageRatePerSecond: avgRate,
					ResetsDetected:       ptrInt(resets),
					FirstTimestamp:       ptrFloat(firstTimestamp),
					LastTimestamp:        ptrFloat(lastTimestamp),
				})
				continue
			}

			floats := make([]float64, len(samples))
			for i, sample := range samples {
				floats[i] = sample[1]
			}
			sort.Float64s(floats)

			minV := floats[0]
			maxV := floats[len(floats)-1]
			sum := 0.0
			for _, v := range floats {
				sum += v
			}
			avg := sum / float64(len(floats))
			p95idx := int(math.Floor(float64(len(floats)) * 0.95))
			if p95idx >= len(floats) {
				p95idx = len(floats) - 1
			}
			p99idx := int(math.Floor(float64(len(floats)) * 0.99))
			if p99idx >= len(floats) {
				p99idx = len(floats) - 1
			}
			p95 := floats[p95idx]
			p99 := floats[p99idx]
			trendDelta := last - first

			finalEntries = append(finalEntries, MetricSummaryEntry{
				Metric:         name,
				Kind:           "gauge",
				Status:         "normal",
				Labels:         labels,
				SampleCount:    len(samples),
				Min:            ptrFloat(minV),
				Max:            ptrFloat(maxV),
				Avg:            ptrFloat(avg),
				P95:            ptrFloat(p95),
				P99:            ptrFloat(p99),
				First:          ptrFloat(first),
				Last:           ptrFloat(last),
				TrendDelta:     ptrFloat(trendDelta),
				FirstTimestamp: ptrFloat(firstTimestamp),
				LastTimestamp:  ptrFloat(lastTimestamp),
			})
		}
	}

	return MetricSummaryResult{
		ResultType:      resultType,
		TotalRawLines:   seriesCount,
		SummarizedCount: len(finalEntries),
		Entries:         finalEntries,
	}
}

func parseSample(sample []any) (float64, float64, bool) {
	if len(sample) < 2 {
		return 0, 0, false
	}

	timestamp, ok := anyToFloat(sample[0])
	if !ok {
		return 0, 0, false
	}
	value, ok := anyToFloat(sample[1])
	if !ok {
		return 0, 0, false
	}
	return timestamp, value, true
}

func anyToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		var f float64
		_, err := fmt.Sscanf(x, "%f", &f)
		return f, err == nil
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func metricLabels(metric map[string]string) map[string]string {
	labels := make(map[string]string)
	for key, value := range metric {
		if key == "__name__" {
			continue
		}
		labels[key] = value
	}
	return labels
}

func metricKind(name string) string {
	if strings.HasSuffix(name, "_total") || strings.HasSuffix(name, "_count") || strings.HasSuffix(name, "_sum") {
		return "counter"
	}
	return "gauge"
}

func counterDeltaAndResets(samples [][2]float64) (float64, int) {
	if len(samples) < 2 {
		return 0, 0
	}
	delta := 0.0
	resets := 0
	for i := 1; i < len(samples); i++ {
		previous := samples[i-1][1]
		current := samples[i][1]
		if current >= previous {
			delta += current - previous
		} else {
			resets++
			delta += current
		}
	}
	return delta, resets
}

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int) *int           { return &v }

func main() {
	typeLong := flag.String("type", "logs", "telemetry type: logs or metrics")
	typeShort := flag.String("t", "", "telemetry type: logs or metrics")
	flag.Parse()

	telemetryType := strings.ToLower(*typeLong)
	if *typeShort != "" {
		telemetryType = strings.ToLower(*typeShort)
	}
	if telemetryType != "logs" && telemetryType != "metrics" {
		telemetryType = "logs"
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(string(input)) == "" {
		return
	}

	if telemetryType == "logs" {
		var response LokiResponse
		if err := json.Unmarshal(input, &response); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing Loki JSON: %v\n", err)
			os.Exit(1)
		}
		result := processLokiResponse(response)
		out, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(out))
		return
	}

	var response MetricResponse
	if err := json.Unmarshal(input, &response); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Prometheus JSON: %v\n", err)
		os.Exit(1)
	}
	result := processMetricsResponse(response)
	out, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
