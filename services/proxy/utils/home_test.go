package utils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name            string
		mockResp        string
		mockErr         error
		expectedStatus  int
		expectedContent string
	}{
		{
			name:            "successful health check",
			mockResp:        "Practicality beats purity.",
			mockErr:         nil,
			expectedStatus:  http.StatusOK,
			expectedContent: "Practicality beats purity.",
		},
		{
			name:            "outbound connection failure",
			mockResp:        "",
			mockErr:         errors.New("connection failed"),
			expectedStatus:  http.StatusOK,
			expectedContent: "Could not fetch Zen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock httpGet
			oldHttpGet := httpGet
			defer func() { httpGet = oldHttpGet }()
			httpGet = func(url string) (*http.Response, error) {
				if tt.mockErr != nil {
					return nil, tt.mockErr
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(tt.mockResp)),
				}, nil
			}

			req := httptest.NewRequest("GET", "/api/health", nil)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HealthHandler)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			var response map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("could not unmarshal response: %v", err)
			}

			if response["status"] != "healthy" {
				t.Errorf("expected status healthy, got %v", response["status"])
			}

			outbound := response["outbound_test"].(map[string]interface{})
			if outbound["content"] != tt.expectedContent {
				t.Errorf("expected zen message %v, got %v", tt.expectedContent, outbound["content"])
			}
		})
	}
}

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name                string
		method              string
		path                string
		expectedStatus      int
		expectedContentType string
		expectedMessage     string
	}{
		{
			name:                "successful request",
			method:              "GET",
			path:                "/",
			expectedStatus:      http.StatusOK,
			expectedContentType: "application/json",
			expectedMessage:     "Welcome to the Observability Hub.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HomeHandler)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if contentType := rr.Header().Get("Content-Type"); contentType != tt.expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, tt.expectedContentType)
			}

			var response map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("could not unmarshal response: %v", err)
			}

			if response["message"] != tt.expectedMessage {
				t.Errorf("handler returned unexpected message: got %v want %v",
					response["message"], tt.expectedMessage)
			}
		})
	}
}
