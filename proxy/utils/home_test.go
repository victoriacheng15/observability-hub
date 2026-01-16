package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
