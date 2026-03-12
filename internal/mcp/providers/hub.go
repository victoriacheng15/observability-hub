package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"observability-hub/internal/telemetry"
)

// CommandRunner defines the interface for executing shell commands.
type CommandRunner interface {
	Run(ctx context.Context, name string, arg ...string) ([]byte, error)
}

// RealCommandRunner is the production implementation.
type RealCommandRunner struct{}

func (r *RealCommandRunner) Run(ctx context.Context, name string, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.CombinedOutput()
}

// HubProvider provides tools for host-level introspection and platform status.
type HubProvider struct {
	runner         CommandRunner
	targetServices []string
}

// NewHubProvider creates a new HubProvider.
func NewHubProvider() *HubProvider {
	return &HubProvider{
		runner: &RealCommandRunner{},
		targetServices: []string{
			"ingestion.service",
			"proxy.service",
			"openbao.service",
			"tailscale-gate.service",
		},
	}
}

// HostResource represents physical resource usage on the host.
type HostResource struct {
	CPUUsage    string `json:"cpu_usage"`
	MemoryTotal string `json:"memory_total"`
	MemoryUsed  string `json:"memory_used"`
	DiskUsage   string `json:"disk_usage"`
	LoadAverage string `json:"load_average"`
}

// ServiceStatus represents the state of a systemd unit.
type ServiceStatus struct {
	Name   string `json:"name"`
	Active string `json:"active"`
	Sub    string `json:"sub"`
	Since  string `json:"since"`
}

// ListHostServices returns the status of target systemd services.
func (p *HubProvider) ListHostServices(ctx context.Context) ([]ServiceStatus, error) {
	var statuses []ServiceStatus

	for _, svc := range p.targetServices {
		out, err := p.runner.Run(ctx, "systemctl", "show", svc, "--property=ActiveState,SubState,ActiveEnterTimestamp")
		if err != nil {
			telemetry.Warn("systemctl_show_failed", "service", svc, "error", err)
			continue
		}

		status := ServiceStatus{Name: svc}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 2 {
				continue
			}
			val := strings.TrimSpace(parts[1])
			switch parts[0] {
			case "ActiveState":
				status.Active = val
			case "SubState":
				status.Sub = val
			case "ActiveEnterTimestamp":
				status.Since = val
			}
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// QueryServiceLogs retrieves journal logs for a specific service since a relative time.
func (p *HubProvider) QueryServiceLogs(ctx context.Context, service string, since string) (string, error) {
	if since == "" {
		since = "5m"
	}

	argSince := fmt.Sprintf("%s ago", since)
	out, err := p.runner.Run(ctx, "journalctl", "-u", service, "--since", argSince, "--no-pager", "-n", "50")
	if err != nil {
		return "", fmt.Errorf("failed to fetch logs for %s: %w", service, err)
	}

	return string(out), nil
}

// InspectHost retrieves physical resource statistics.
func (p *HubProvider) InspectHost(ctx context.Context) (*HostResource, error) {
	res := &HostResource{}

	// Load Average
	loadOut, _ := p.runner.Run(ctx, "uptime")
	res.LoadAverage = strings.TrimSpace(string(loadOut))

	// Memory (free -h)
	memOut, _ := p.runner.Run(ctx, "free", "-h")
	lines := strings.Split(string(memOut), "\n")
	if len(lines) > 1 {
		fields := strings.Fields(lines[1]) // Mem row
		if len(fields) > 2 {
			res.MemoryTotal = fields[1]
			res.MemoryUsed = fields[2]
		}
	}

	// Disk (df -h /)
	diskOut, _ := p.runner.Run(ctx, "df", "-h", "/")
	dLines := strings.Split(string(diskOut), "\n")
	if len(dLines) > 1 {
		dFields := strings.Fields(dLines[1])
		if len(dFields) > 4 {
			res.DiskUsage = dFields[4]
		}
	}

	return res, nil
}

// InspectPlatform returns an executive summary of the entire hub.
func (p *HubProvider) InspectPlatform(ctx context.Context) (map[string]interface{}, error) {
	summary := make(map[string]interface{})
	summary["timestamp"] = time.Now().Format(time.RFC3339)
	summary["node"] = "server2"

	if _, err := p.runner.Run(ctx, "kubectl", "get", "nodes"); err != nil {
		summary["k3s_status"] = "unreachable"
	} else {
		summary["k3s_status"] = "healthy"
	}

	services, _ := p.ListHostServices(ctx)
	healthyCount := 0
	for _, s := range services {
		// Count as healthy if:
		// 1. Service is 'active' (running)
		// 2. Service is 'inactive' but the substate is 'dead' (successfully completed oneshot)
		if s.Active == "active" || (s.Active == "inactive" && s.Sub == "dead") {
			healthyCount++
		}
	}
	summary["host_services_healthy"] = fmt.Sprintf("%d/%d", healthyCount, len(p.targetServices))

	return summary, nil
}
