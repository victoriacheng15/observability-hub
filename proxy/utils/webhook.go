package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// Payload represents the payload from GitHub webhook push events
type Payload struct {
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Ref string `json:"ref"`
}

// WebhookHandler handles GitHub push event webhooks to trigger GitOps sync
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		slog.Error("GITHUB_WEBHOOK_SECRET is not set")
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		slog.Warn("Missing X-Hub-Signature-256 header")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if !verifySignature(body, signature, secret) {
		slog.Warn("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("Failed to parse webhook payload", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Only trigger on main branch pushes
	if payload.Ref != "refs/heads/main" {
		slog.Info("Ignored webhook for non-main branch", "ref", payload.Ref)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ignored non-main branch"))
		return
	}

	repoName := payload.Repository.Name
	if repoName == "" {
		slog.Warn("Repository name missing in payload")
		http.Error(w, "Repository name missing", http.StatusBadRequest)
		return
	}

	// Security Gate: Validate repo name or ensure it's a known repository
	// For now, we'll proceed as it's passed to gitops_sync.sh which should handle it
	go func(repo string) {
		slog.Info("Triggering GitOps sync via webhook", "repo", repo)
		// We use an absolute path to the script for reliability
		cmd := exec.Command("/home/server/software/observability-hub/scripts/gitops_sync.sh", repo)
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("GitOps sync execution failed", "repo", repo, "error", err, "output", string(output))
		} else {
			slog.Info("GitOps sync execution successful", "repo", repo, "output", string(output))
		}
	}(repoName)

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Sync triggered for %s", repoName)
}

func verifySignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature[7:]), []byte(expectedMAC))
}
