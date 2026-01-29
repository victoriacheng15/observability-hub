package utils

import (
	"encoding/json"
	"io"
	"net/http"
)

var httpGet = http.Get

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Welcome to the Observability Hub."})
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch from an external API (GitHub Zen) to test outbound connectivity
	resp, err := httpGet("https://api.github.com/zen")
	zenMessage := "Could not fetch Zen"
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		zenMessage = string(body)
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
