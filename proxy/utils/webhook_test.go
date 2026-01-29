package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWebhookHandler(t *testing.T) {
	testSecret := "test-secret-123"

	// Helper to generate valid signatures
	generateSignature := func(secret string, body []byte) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		return "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	tests := map[string]struct {
		method         string
		eventType      string
		envSecret      string
		body           map[string]interface{}
		headerSig      string // Optional override
		expectedStatus int
		expectedBody   string
	}{
		"method_not_allowed": {
			method:         http.MethodGet,
			envSecret:      testSecret,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		"ping_event": {
			method:         http.MethodPost,
			eventType:      "ping",
			envSecret:      testSecret,
			expectedStatus: http.StatusOK,
			expectedBody:   "Pong",
		},
		"missing_secret_in_env": {
			method:         http.MethodPost,
			eventType:      "push",
			envSecret:      "",
			expectedStatus: http.StatusInternalServerError,
		},
		"missing_signature_header": {
			method:         http.MethodPost,
			eventType:      "push",
			envSecret:      testSecret,
			expectedStatus: http.StatusUnauthorized,
		},
		"invalid_signature": {
			method:    http.MethodPost,
			eventType: "push",
			envSecret: testSecret,
			body: map[string]interface{}{
				"ref": "refs/heads/main",
			},
			headerSig:      "sha256=invalidsignature",
			expectedStatus: http.StatusUnauthorized,
		},
		"ignored_branch_dev": {
			method:    http.MethodPost,
			eventType: "push",
			envSecret: testSecret,
			body: map[string]interface{}{
				"ref": "refs/heads/dev",
				"repository": map[string]string{
					"name": "test-repo",
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Ignored: Not a push to main or closed PR to main",
		},
		"success_main_branch": {
			method:    http.MethodPost,
			eventType: "push",
			envSecret: testSecret,
			body: map[string]interface{}{
				"ref": "refs/heads/main",
				"repository": map[string]string{
					"name": "test-repo",
				},
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "Sync triggered for test-repo",
		},
		"success_pr_merge": {
			method:    http.MethodPost,
			eventType: "pull_request",
			envSecret: testSecret,
			body: map[string]interface{}{
				"action": "closed",
				"pull_request": map[string]interface{}{
					"merged": true,
					"base": map[string]string{
						"ref": "main",
					},
				},
				"repository": map[string]string{
					"name": "test-repo",
				},
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "Sync triggered for test-repo",
		},
		"success_pr_closed_not_merged": {
			method:    http.MethodPost,
			eventType: "pull_request",
			envSecret: testSecret,
			body: map[string]interface{}{
				"action": "closed",
				"pull_request": map[string]interface{}{
					"merged": false,
					"base": map[string]string{
						"ref": "main",
					},
				},
				"repository": map[string]string{
					"name": "test-repo",
				},
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "Sync triggered for test-repo",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup Environment
			if tt.envSecret != "" {
				os.Setenv("GITHUB_WEBHOOK_SECRET", tt.envSecret)
			} else {
				os.Unsetenv("GITHUB_WEBHOOK_SECRET")
			}
			// Cleanup after test
			defer os.Unsetenv("GITHUB_WEBHOOK_SECRET")

			// Prepare Body
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/api/webhook/gitops", bytes.NewReader(bodyBytes))

			// Prepare Headers
			if tt.eventType != "" {
				req.Header.Set("X-GitHub-Event", tt.eventType)
			}

			if tt.method == http.MethodPost && tt.envSecret != "" {
				if tt.headerSig != "" {
					req.Header.Set("X-Hub-Signature-256", tt.headerSig)
				} else if tt.body != nil {
					// Auto-generate valid signature if not testing invalid one
					req.Header.Set("X-Hub-Signature-256", generateSignature(tt.envSecret, bodyBytes))
				}
			}

			w := httptest.NewRecorder()
			WebhookHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}
