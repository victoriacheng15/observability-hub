package utils

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type SyntheticPayload struct {
	Region      string `json:"region"`
	Timezone    string `json:"timezone"`
	Device      string `json:"device"`
	NetworkType string `json:"network_type"`
}

func SyntheticTraceHandler(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())

	// Extract synthetic ID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/trace/synthetic/"), "/")
	syntheticID := parts[0]

	trafficMode := r.Header.Get("X-Traffic-Mode")
	if trafficMode == "" {
		trafficMode = "unknown"
	}

	// 1. Attributes
	span.SetAttributes(
		attribute.String("app.synthetic_id", syntheticID),
		attribute.String("app.traffic_mode", trafficMode),
	)

	// 2. Decode Payload
	var payload SyntheticPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
		// Capture raw payload as event
		payloadJSON, _ := json.Marshal(payload)
		span.AddEvent("request.payload.received", trace.WithAttributes(
			attribute.String("payload.body", string(payloadJSON)),
		))

		// Business Attributes
		span.SetAttributes(
			attribute.String("app.business.region", payload.Region),
			attribute.String("app.business.timezone", payload.Timezone),
			attribute.String("app.business.device", payload.Device),
			attribute.String("app.business.network_type", payload.NetworkType),
		)
	}

	// 3. Latency Simulation (Jitter)
	latencyTarget := rand.Intn(46) + 5 // 5ms to 50ms
	span.SetAttributes(attribute.Int("app.latency_target_ms", latencyTarget))

	span.AddEvent("processing.simulated_delay", trace.WithAttributes(
		attribute.Int("delay_ms", latencyTarget),
	))

	time.Sleep(time.Duration(latencyTarget) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"synthetic_id": syntheticID,
		"latency_ms":   latencyTarget,
	})
}
