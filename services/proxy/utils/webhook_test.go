package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

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
			expectedBody:   "Method not allowed",
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
			expectedBody:   "Server configuration error",
		},
		"missing_signature_header": {
			method:         http.MethodPost,
			eventType:      "push",
			envSecret:      testSecret,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Missing signature",
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
			expectedBody:   "Invalid signature",
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
		"missing_event_header_ignored": {
			method:    http.MethodPost,
			eventType: "",
			envSecret: testSecret,
			body: map[string]interface{}{
				"ref": "refs/heads/main",
				"repository": map[string]string{
					"name": "test-repo",
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Ignored: Not a push to main or closed PR to main",
		},
		"invalid_json": {
			method:    http.MethodPost,
			eventType: "push",
			envSecret: testSecret,
			headerSig: "sha256=invalidbutformatted",
		},
		"missing_repo_name": {
			method:    http.MethodPost,
			eventType: "push",
			envSecret: testSecret,
			body: map[string]interface{}{
				"ref":        "refs/heads/main",
				"repository": map[string]string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Repository name missing",
		},
		"invalid_signature_prefix": {
			method:    http.MethodPost,
			eventType: "push",
			envSecret: testSecret,
			body: map[string]interface{}{
				"ref": "refs/heads/main",
			},
			headerSig:      "plain-text-signature",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid signature",
		},
		"body_read_error": {
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to read body",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if name == "invalid_json" {
				os.Setenv("GITHUB_WEBHOOK_SECRET", testSecret)
				defer os.Unsetenv("GITHUB_WEBHOOK_SECRET")
				body := []byte("{invalid-json}")
				req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
				req.Header.Set("X-GitHub-Event", "push")
				req.Header.Set("X-Hub-Signature-256", generateSignature(testSecret, body))
				w := httptest.NewRecorder()
				WebhookHandler(w, req)
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
				}
				if !bytes.Contains(w.Body.Bytes(), []byte("Invalid JSON")) {
					t.Errorf("Expected body to contain %q, got %q", "Invalid JSON", w.Body.String())
				}
				return
			}

			if name == "body_read_error" {
				os.Setenv("GITHUB_WEBHOOK_SECRET", testSecret)
				defer os.Unsetenv("GITHUB_WEBHOOK_SECRET")
				req := httptest.NewRequest("POST", "/", &errorReader{})
				req.Header.Set("X-GitHub-Event", "push")
				req.Header.Set("X-Hub-Signature-256", "sha256=any")
				w := httptest.NewRecorder()
				WebhookHandler(w, req)
				if w.Code != http.StatusInternalServerError {
					t.Errorf("Expected 500 for body read error, got %d", w.Code)
				}
				if !bytes.Contains(w.Body.Bytes(), []byte("Failed to read body")) {
					t.Errorf("Expected body to contain %q, got %q", "Failed to read body", w.Body.String())
				}
				return
			}

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
