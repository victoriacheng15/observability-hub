package providers

import (
	"context"
	"fmt"
	"strings"

	mcpcommand "observability-hub/internal/mcp/command"
	"observability-hub/internal/telemetry"
)

// NetworkProvider provides network observability tools backed by Cilium Hubble.
type NetworkProvider struct {
	runner mcpcommand.Runner
}

// NewNetworkProvider creates a NetworkProvider.
func NewNetworkProvider() *NetworkProvider {
	return &NetworkProvider{runner: mcpcommand.NewExecutor([]string{"kubectl"})}
}

// QueryHubbleFlows retrieves real-time flow data from Hubble Relay.
// Since the hubble CLI is not on the host, it execs into the Cilium agent pod.
func (p *NetworkProvider) QueryHubbleFlows(ctx context.Context, namespace, pod, fromPod, toPod, protocol, verdict, httpStatus, httpMethod, httpPath, reserved string, port, toPort, last int) (string, error) {
	if last <= 0 {
		last = 20
	}
	if last > 100 {
		last = 100
	}

	hubbleArgs := []string{"observe", "--last", fmt.Sprintf("%d", last), "--output", "json"}

	if namespace != "" {
		hubbleArgs = append(hubbleArgs, "--namespace", namespace)
	}
	if pod != "" {
		hubbleArgs = append(hubbleArgs, "--pod", pod)
	}
	if reserved != "" {
		hubbleArgs = append(hubbleArgs, "--label", fmt.Sprintf("reserved:%s", reserved))
	}
	if fromPod != "" {
		hubbleArgs = append(hubbleArgs, "--from-pod", fromPod)
	}
	if toPod != "" {
		hubbleArgs = append(hubbleArgs, "--to-pod", toPod)
	}
	if protocol != "" {
		hubbleArgs = append(hubbleArgs, "--protocol", protocol)
	}
	if port > 0 {
		hubbleArgs = append(hubbleArgs, "--port", fmt.Sprintf("%d", port))
	}
	if toPort > 0 {
		hubbleArgs = append(hubbleArgs, "--to-port", fmt.Sprintf("%d", toPort))
	}
	if verdict != "" {
		hubbleArgs = append(hubbleArgs, "--verdict", verdict)
	}
	if httpStatus != "" {
		hubbleArgs = append(hubbleArgs, "--http-status", httpStatus)
	}
	if httpMethod != "" {
		hubbleArgs = append(hubbleArgs, "--http-method", httpMethod)
	}
	if httpPath != "" {
		hubbleArgs = append(hubbleArgs, "--http-path", httpPath)
	}

	args := []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock"}
	args = append(args, hubbleArgs...)

	result, err := p.runner.Run(ctx, mcpcommand.Request{Name: "kubectl", Args: args})
	if err != nil {
		output := string(result.Stdout) + string(result.Stderr)
		telemetry.Error("hubble observe via kubectl failed", "error", err, "output", output)
		return "", fmt.Errorf("hubble observe failed: %w", err)
	}

	lines := strings.Split(string(result.Stdout), "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Defaulted container") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}

	return strings.Join(cleaned, "\n"), nil
}
