package utils

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"telemetry"
	"time"
)

var webhookTracer = telemetry.GetTracer("proxy.webhook")
var webhookMeter = telemetry.GetMeter("proxy.webhook")

var repoLocks sync.Map

var (
	webhookMetricsOnce      sync.Once
	webhookMetricsReady     bool
	webhookReceivedTotal    telemetry.Int64Counter
	webhookErrorsTotal      telemetry.Int64Counter
	webhookSyncDurationMsec telemetry.Int64Histogram
)

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

func ensureWebhookMetrics() {
	webhookMetricsOnce.Do(func() {
		var err error
		webhookReceivedTotal, err = telemetry.NewInt64Counter(
			webhookMeter,
			"proxy.webhook.received.total",
			"Total webhook requests received",
		)
		if err != nil {
			telemetry.Warn("webhook_metric_init_failed", "metric", "proxy.webhook.received.total", "error", err)
			return
		}

		webhookErrorsTotal, err = telemetry.NewInt64Counter(
			webhookMeter,
			"proxy.webhook.errors.total",
			"Total webhook request errors",
		)
		if err != nil {
			telemetry.Warn("webhook_metric_init_failed", "metric", "proxy.webhook.errors.total", "error", err)
			return
		}

		webhookSyncDurationMsec, err = telemetry.NewInt64Histogram(
			webhookMeter,
			"proxy.webhook.sync.duration.ms",
			"Webhook request duration in milliseconds",
			"ms",
		)
		if err != nil {
			telemetry.Warn("webhook_metric_init_failed", "metric", "proxy.webhook.sync.duration.ms", "error", err)
			return
		}

		webhookMetricsReady = true
	})
}

// WebhookHandler handles GitHub push event webhooks to trigger GitOps sync
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ensureWebhookMetrics()

	ctx, span := webhookTracer.Start(r.Context(), "handler.webhook")
	defer span.End()

	// Advanced: Peer IP for analysis
	span.SetAttributes(telemetry.StringAttribute("net.peer.ip", r.RemoteAddr))

	eventType := r.Header.Get("X-GitHub-Event")
	span.SetAttributes(telemetry.StringAttribute("github.event", eventType))

	metricAttrs := []telemetry.Attribute{
		telemetry.StringAttribute("github.event", eventType),
	}
	defer func() {
		if webhookMetricsReady {
			durationMs := time.Since(start).Milliseconds()
			telemetry.RecordInt64Histogram(ctx, webhookSyncDurationMsec, durationMs, metricAttrs...)
		}
	}()
	if webhookMetricsReady {
		telemetry.AddInt64Counter(ctx, webhookReceivedTotal, 1, metricAttrs...)
	}

	if r.Method != http.MethodPost {
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "method_not_allowed")
		span.SetAttributes(telemetry.BoolAttribute("error", true))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	telemetry.Info("webhook_received", "event", eventType)

	// Gracefully handle ping events
	if eventType == "ping" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Pong"))
		return
	}

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "missing_secret")
		span.SetAttributes(telemetry.BoolAttribute("error", true))
		telemetry.Error("webhook_secret_missing")
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "missing_signature")
		span.SetAttributes(telemetry.BoolAttribute("error", true))
		telemetry.Warn("webhook_signature_missing")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "body_read_failed")
		span.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		telemetry.Error("webhook_body_read_failed", "error", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	if !verifySignature(body, signature, secret) {
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "invalid_signature")
		span.SetAttributes(telemetry.BoolAttribute("error", true))
		telemetry.Warn("webhook_signature_invalid")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "payload_invalid")
		span.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		telemetry.Error("webhook_payload_invalid", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Newbie level: keep minimal, non-sensitive attributes only.
	span.SetAttributes(
		telemetry.StringAttribute("github.repo", payload.Repository.FullName),
		telemetry.StringAttribute("github.ref", payload.Ref),
		telemetry.StringAttribute("github.action", payload.Action),
	)

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
		telemetry.Info("webhook_ignored",
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
		if webhookMetricsReady {
			telemetry.AddInt64Counter(ctx, webhookErrorsTotal, 1, metricAttrs...)
		}
		span.SetStatus(telemetry.CodeError, "missing_repo_name")
		span.SetAttributes(telemetry.BoolAttribute("error", true))
		telemetry.Warn("webhook_repo_name_missing", "event", eventType)
		http.Error(w, "Repository name missing", http.StatusBadRequest)
		return
	}

	go func(repo string, merged bool, parentCtx context.Context) {
		// Acquire lock for this specific repository
		val, _ := repoLocks.LoadOrStore(repo, &sync.Mutex{})
		mu := val.(*sync.Mutex)

		mu.Lock()

		defer mu.Unlock()

		_, syncSpan := webhookTracer.Start(parentCtx, "webhook.gitops")
		syncSpan.SetAttributes(
			telemetry.StringAttribute("github.repo", repo),
			telemetry.StringAttribute("github.event", eventType),
			telemetry.BoolAttribute("github.merged", merged),
		)
		defer syncSpan.End()

		telemetry.Info("webhook_sync_triggered",
			"repo", repo,
			"event", eventType,
			"merged", merged,
		)
		// We use an absolute path to the script for reliability
		cmd := exec.Command("/home/server/software/observability-hub/scripts/gitops_sync.sh", repo)
		output, err := cmd.CombinedOutput()
		if err != nil {
			syncSpan.RecordError(err)
			syncSpan.SetStatus(telemetry.CodeError, "sync failed")
			telemetry.Error("webhook_sync_failed", "repo", repo, "error", err, "output", string(output))
		} else {
			telemetry.Info("webhook_sync_success", "repo", repo, "output", string(output))
		}
	}(repoName, isMerged, ctx)

	w.WriteHeader(http.StatusAccepted)
	telemetry.Info("webhook_processed", "repo", repoName, "event", eventType, "status", http.StatusAccepted)
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
