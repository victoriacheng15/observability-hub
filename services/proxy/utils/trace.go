package utils

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"telemetry"
	"time"
)

var syntheticTracer = telemetry.GetTracer("proxy/synthetic")

type SyntheticPayload struct {
	Region      string `json:"region"`
	Timezone    string `json:"timezone"`
	Device      string `json:"device"`
	NetworkType string `json:"network_type"`
}

func SyntheticTraceHandler(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromContext(r.Context())

	// Extract synthetic ID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/trace/synthetic/"), "/")
	syntheticID := parts[0]

	trafficMode := r.Header.Get("X-Traffic-Mode")
	if trafficMode == "" {
		trafficMode = "unknown"
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
	}

	// 3. Latency Simulation (Jitter)
	latencyTarget := rand.Intn(46) + 5 // 5ms to 50ms
	span.SetAttributes(telemetry.IntAttribute("app.latency_target_ms", latencyTarget))

	span.AddEvent("processing.simulated_delay", telemetry.WithEventAttributes(
		telemetry.IntAttribute("delay_ms", latencyTarget),
	))

	time.Sleep(time.Duration(latencyTarget) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"synthetic_id": syntheticID,
		"latency_ms":   latencyTarget,
	})
}
