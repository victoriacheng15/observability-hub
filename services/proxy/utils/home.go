package utils

import (
	"encoding/json"
	"io"
	"net/http"
	telemetry "telemetry-next"
)

var homeTracer = telemetry.GetTracer("proxy/home")

var httpGet = http.Get

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Welcome to the Observability Hub."})
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromContext(r.Context())

	// 1. Fetch from an external API (GitHub Zen) to test outbound connectivity
	resp, err := httpGet("https://api.github.com/zen")
	zenMessage := "Could not fetch Zen"
	if err == nil {
		defer resp.Body.Close()
		zenBody, _ := io.ReadAll(resp.Body)
		zenMessage = string(zenBody)

		// Record the "Payload" (Response from GitHub) in the trace
		span.SetAttributes(telemetry.StringAttribute("app.outbound.zen_message", zenMessage))
		span.AddEvent("outbound.response_received", telemetry.WithEventAttributes(
			telemetry.StringAttribute("outbound.source", "github_zen"),
			telemetry.StringAttribute("payload.body", zenMessage),
		))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"outbound_test": map[string]string{
			"source":  "https://api.github.com/zen",
			"content": zenMessage,
		},
	})
}
