package utils

import (
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"telemetry"
	"time"
)

var syntheticTracer = telemetry.GetTracer("proxy/synthetic")
var syntheticMeter = telemetry.GetMeter("proxy/synthetic")

var (
	syntheticMetricsOnce         sync.Once
	syntheticMetricsReady        bool
	syntheticRequestTotal        telemetry.Int64Counter
	syntheticRequestErrorsTotal  telemetry.Int64Counter
	syntheticRequestDurationMsec telemetry.Int64Histogram
)

func ensureSyntheticMetrics() {
	syntheticMetricsOnce.Do(func() {
		var err error
		syntheticRequestTotal, err = telemetry.NewInt64Counter(
			syntheticMeter,
			"proxy.synthetic.request.total",
			"Total synthetic trace requests received",
		)
		if err != nil {
			slog.Warn("synthetic_metric_init_failed", "metric", "proxy.synthetic.request.total", "error", err)
			return
		}

		syntheticRequestErrorsTotal, err = telemetry.NewInt64Counter(
			syntheticMeter,
			"proxy.synthetic.request.errors.total",
			"Total synthetic trace request errors",
		)
		if err != nil {
			slog.Warn("synthetic_metric_init_failed", "metric", "proxy.synthetic.request.errors.total", "error", err)
			return
		}

		syntheticRequestDurationMsec, err = telemetry.NewInt64Histogram(
			syntheticMeter,
			"proxy.synthetic.request.duration.ms",
			"Synthetic trace request duration in milliseconds",
			"ms",
		)
		if err != nil {
			slog.Warn("synthetic_metric_init_failed", "metric", "proxy.synthetic.request.duration.ms", "error", err)
			return
		}

		syntheticMetricsReady = true
	})
}

type SyntheticPayload struct {
	Region      string `json:"region"`
	Timezone    string `json:"timezone"`
	Device      string `json:"device"`
	NetworkType string `json:"network_type"`
}

func SyntheticTraceHandler(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromContext(r.Context())
	start := time.Now()
	ensureSyntheticMetrics()

	// Extract synthetic ID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/trace/synthetic/"), "/")
	syntheticID := parts[0]

	trafficMode := r.Header.Get("X-Traffic-Mode")
	if trafficMode == "" {
		trafficMode = "unknown"
	}
	metricAttrs := []telemetry.Attribute{
		telemetry.StringAttribute("app.traffic_mode", trafficMode),
	}
	if syntheticMetricsReady {
		telemetry.AddInt64Counter(r.Context(), syntheticRequestTotal, 1, metricAttrs...)
	}

	_, span = syntheticTracer.Start(r.Context(), "handler.synthetic_trace")
	defer span.End()

	// 1. Attributes
	span.SetAttributes(
		telemetry.StringAttribute("app.synthetic_id", syntheticID),
		telemetry.StringAttribute("app.traffic_mode", trafficMode),
	)

	// 2. Decode Payload
	var payload SyntheticPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
		// Capture raw payload as event
		payloadJSON, _ := json.Marshal(payload)
		span.AddEvent("request.payload.received", telemetry.WithEventAttributes(
			telemetry.StringAttribute("payload.body", string(payloadJSON)),
		))

		// Business Attributes
		span.SetAttributes(
			telemetry.StringAttribute("app.business.region", payload.Region),
			telemetry.StringAttribute("app.business.timezone", payload.Timezone),
			telemetry.StringAttribute("app.business.device", payload.Device),
			telemetry.StringAttribute("app.business.network_type", payload.NetworkType),
		)
		slog.Info("synthetic_trace_payload_received",
			"synthetic_id", syntheticID,
			"traffic_mode", trafficMode,
			"region", payload.Region,
			"device", payload.Device,
			"network_type", payload.NetworkType,
		)
	} else if err != io.EOF {
		span.SetStatus(telemetry.CodeError, "payload_decode_failed")
		span.AddEvent("request.payload.decode_failed", telemetry.WithEventAttributes(
			telemetry.StringAttribute("error", err.Error()),
		))
		if syntheticMetricsReady {
			telemetry.AddInt64Counter(r.Context(), syntheticRequestErrorsTotal, 1, metricAttrs...)
		}
		slog.Warn("synthetic_trace_payload_decode_failed",
			"synthetic_id", syntheticID,
			"traffic_mode", trafficMode,
			"error", err,
		)
	}

	// 3. Latency Simulation (Jitter)
	latencyTarget := rand.Intn(46) + 5 // 5ms to 50ms
	span.SetAttributes(telemetry.IntAttribute("app.latency_target_ms", latencyTarget))

	span.AddEvent("processing.simulated_delay", telemetry.WithEventAttributes(
		telemetry.IntAttribute("delay_ms", latencyTarget),
	))

	time.Sleep(time.Duration(latencyTarget) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"synthetic_id": syntheticID,
		"latency_ms":   latencyTarget,
	}); err != nil {
		span.SetStatus(telemetry.CodeError, "response_encode_failed")
		if syntheticMetricsReady {
			telemetry.AddInt64Counter(r.Context(), syntheticRequestErrorsTotal, 1, metricAttrs...)
		}
		slog.Error("synthetic_trace_response_encode_failed",
			"synthetic_id", syntheticID,
			"traffic_mode", trafficMode,
			"error", err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	durationMs := time.Since(start).Milliseconds()
	if syntheticMetricsReady {
		telemetry.RecordInt64Histogram(r.Context(), syntheticRequestDurationMsec, durationMs, metricAttrs...)
	}
	slog.Info("synthetic_trace_processed",
		"synthetic_id", syntheticID,
		"traffic_mode", trafficMode,
		"latency_target_ms", latencyTarget,
		"duration_ms", durationMs,
	)
}
