package utils

import (
	"context"
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
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var repoLocks sync.Map

type Payload struct {
	Ref        string `json:"ref"`
	Action     string `json:"action"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Merged bool `json:"merged"`
		Base   struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
}

// WebhookHandler handles GitHub push event webhooks to trigger GitOps sync
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	slog.Info("Received webhook", "event", eventType)

	// Gracefully handle ping events
	if eventType == "ping" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Pong"))
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

	ctx := r.Context()

	_, verifySpan := tracer.Start(ctx, "webhook.verify_signature")
	if !verifySignature(body, signature, secret) {
		verifySpan.SetStatus(codes.Error, "invalid signature")
		verifySpan.End()
		slog.Warn("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}
	verifySpan.End()

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("Failed to parse webhook payload", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Enrich the root span with metadata and raw payload
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("github.event", eventType),
		attribute.String("github.repo", payload.Repository.FullName),
		attribute.String("github.ref", payload.Ref),
		attribute.String("github.action", payload.Action),
	)
	span.AddEvent("webhook.payload_received", trace.WithAttributes(
		attribute.String("payload.raw", string(body)),
	))

	shouldTrigger := false

	// Case 1: Push to main
	if eventType == "push" && payload.Ref == "refs/heads/main" {
		shouldTrigger = true
	}

	// Case 2: PR closed on main (Merged flag is preferred but optional for redundancy)
	if eventType == "pull_request" && payload.Action == "closed" && payload.PullRequest.Base.Ref == "main" {
		shouldTrigger = true
	}

	if !shouldTrigger {
		slog.Info("Ignored webhook",
			"repo", payload.Repository.Name,
			"event", eventType,
			"ref", payload.Ref,
			"action", payload.Action,
			"merged", payload.PullRequest.Merged,
		)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ignored: Not a push to main or closed PR to main"))
		return
	}

	repoName := payload.Repository.Name
	isMerged := payload.PullRequest.Merged
	if repoName == "" {
		slog.Warn("Repository name missing in payload", "event", eventType)
		http.Error(w, "Repository name missing", http.StatusBadRequest)
		return
	}

	go func(repo string, merged bool, parentCtx context.Context) {
		// Acquire lock for this specific repository
		val, _ := repoLocks.LoadOrStore(repo, &sync.Mutex{})
		mu := val.(*sync.Mutex)

		_, waitSpan := tracer.Start(parentCtx, "webhook.wait_for_lock")
		waitSpan.SetAttributes(attribute.String("github.repo", repo))
		mu.Lock()
		waitSpan.End()

		defer mu.Unlock()

		_, syncSpan := tracer.Start(parentCtx, "webhook.gitops_sync")
		syncSpan.SetAttributes(
			attribute.String("github.repo", repo),
			attribute.String("github.event", eventType),
			attribute.Bool("github.merged", merged),
		)
		defer syncSpan.End()

		slog.Info("Triggering GitOps sync via webhook",
			"repo", repo,
			"event", eventType,
			"merged", merged,
		)
		// We use an absolute path to the script for reliability
		cmd := exec.Command("/home/server/software/observability-hub/scripts/gitops_sync.sh", repo)
		output, err := cmd.CombinedOutput()
		if err != nil {
			syncSpan.RecordError(err)
			syncSpan.SetStatus(codes.Error, "sync failed")
			slog.Error("GitOps sync execution failed", "repo", repo, "error", err, "output", string(output))
		} else {
			slog.Info("GitOps sync execution successful", "repo", repo, "output", string(output))
		}
	}(repoName, isMerged, ctx)

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
