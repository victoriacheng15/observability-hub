package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner defines the interface for executing shell commands.
type CommandRunner interface {
	Run(ctx context.Context, name string, arg ...string) ([]byte, error)
}

// RealCommandRunner is the production implementation that actually runs commands.
type RealCommandRunner struct{}

func (r *RealCommandRunner) Run(ctx context.Context, name string, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.CombinedOutput()
}

// Global runner that can be swapped in tests.
var runner CommandRunner = &RealCommandRunner{}

// FunnelStatus represents the current public-facing state of the Tailscale Funnel.
type FunnelStatus struct {
	Active  bool      `json:"active"`
	Target  string    `json:"target"`
	Port    string    `json:"port"`
	Fetched time.Time `json:"fetched_at"`
}

// GetFunnelStatus executes 'tailscale funnel status' to get the definitive source of truth.
func GetFunnelStatus(ctx context.Context) (*FunnelStatus, error) {
	output, err := runner.Run(ctx, "tailscale", "funnel", "status")
	if err != nil {
		// If 'funnel status' returns an error, it usually means it's off or not configured.
		return &FunnelStatus{Active: false, Fetched: time.Now()}, nil
	}

	outStr := string(output)
	status := &FunnelStatus{
		Active:  strings.Contains(outStr, "(Funnel on)"),
		Fetched: time.Now(),
	}

	// Basic parsing for the target URL if active.
	if status.Active {
		lines := strings.Split(outStr, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "https://") {
				status.Target = strings.Fields(trimmed)[0]
				break
			}
		}
	}

	return status, nil
}

// GetTailscaleStatus executes the Tailscale CLI to fetch current node health via JSON.
func GetTailscaleStatus(ctx context.Context) (map[string]interface{}, error) {
	output, err := runner.Run(ctx, "tailscale", "status", "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to run tailscale status --json: %w", err)
	}

	var status map[string]interface{}
	if err := json.Unmarshal(output, &status); err != nil {
		return nil, fmt.Errorf("failed to decode tailscale status json: %w", err)
	}

	return status, nil
}
