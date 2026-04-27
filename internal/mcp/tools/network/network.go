package network

import (
	"context"
	"fmt"

	toolvalidation "observability-hub/internal/mcp/tools"
)

const (
	maxFlowsLast       = 100
	maxHTTPPathLength  = 256
	maxHTTPStatusToken = 16
	maxNetworkTokenLen = 32
)

// ObserveNetworkFlowsInput is the input for the observe_network_flows tool.
type ObserveNetworkFlowsInput struct {
	// Filter by namespace (source or destination).
	Namespace string `json:"namespace,omitempty"`
	// Filter by pod (source or destination).
	Pod string `json:"pod,omitempty"`
	// Filter by source pod ([namespace/]<pod-name>).
	FromPod string `json:"from_pod,omitempty"`
	// Filter by destination pod ([namespace/]<pod-name>).
	ToPod string `json:"to_pod,omitempty"`
	// Filter by protocol (e.g. "tcp", "udp", "http").
	Protocol string `json:"protocol,omitempty"`
	// Filter by port (source or destination).
	Port int `json:"port,omitempty"`
	// Filter by destination port.
	ToPort int `json:"to_port,omitempty"`
	// Filter by verdict (FORWARDED, DROPPED).
	Verdict string `json:"verdict,omitempty"`
	// Filter by HTTP status code prefix (e.g. "404", "5+").
	HTTPStatus string `json:"http_status,omitempty"`
	// Filter by HTTP method (e.g. "GET", "POST").
	HTTPMethod string `json:"http_method,omitempty"`
	// Filter by HTTP path regex.
	HTTPPath string `json:"http_path,omitempty"`
	// Filter by reserved entity (e.g. "host", "world").
	Reserved string `json:"reserved,omitempty"`
	// Number of recent flows (default 20, max 100).
	Last int `json:"last,omitempty"`
}

// ObserveNetworkFlowsHandler handles real-time network flow observation via Hubble.
type ObserveNetworkFlowsHandler struct {
	queryFn func(ctx context.Context, namespace, pod, fromPod, toPod, protocol, verdict, httpStatus, httpMethod, httpPath, reserved string, port, toPort, last int) (string, error)
}

func NewObserveNetworkFlowsHandler(fn func(ctx context.Context, namespace, pod, fromPod, toPod, protocol, verdict, httpStatus, httpMethod, httpPath, reserved string, port, toPort, last int) (string, error)) *ObserveNetworkFlowsHandler {
	return &ObserveNetworkFlowsHandler{queryFn: fn}
}

func (h *ObserveNetworkFlowsHandler) Execute(ctx context.Context, input ObserveNetworkFlowsInput) (interface{}, error) {
	if err := validateObserveNetworkFlowsInput(input); err != nil {
		return nil, err
	}
	return h.queryFn(ctx,
		input.Namespace,
		input.Pod,
		input.FromPod,
		input.ToPod,
		input.Protocol,
		input.Verdict,
		input.HTTPStatus,
		input.HTTPMethod,
		input.HTTPPath,
		input.Reserved,
		input.Port,
		input.ToPort,
		input.Last,
	)
}

func validateObserveNetworkFlowsInput(input ObserveNetworkFlowsInput) error {
	if err := toolvalidation.OptionalDNS1123Label("namespace", input.Namespace); err != nil {
		return err
	}
	if err := toolvalidation.OptionalPodRef("pod", input.Pod); err != nil {
		return err
	}
	if err := toolvalidation.OptionalPodRef("from_pod", input.FromPod); err != nil {
		return err
	}
	if err := toolvalidation.OptionalPodRef("to_pod", input.ToPod); err != nil {
		return err
	}
	if err := toolvalidation.OptionalSafeToken("protocol", input.Protocol, maxNetworkTokenLen); err != nil {
		return err
	}
	if err := toolvalidation.OptionalSafeToken("verdict", input.Verdict, maxNetworkTokenLen); err != nil {
		return err
	}
	if err := toolvalidation.OptionalSafeToken("http_status", input.HTTPStatus, maxHTTPStatusToken); err != nil {
		return err
	}
	if err := toolvalidation.OptionalSafeToken("http_method", input.HTTPMethod, maxNetworkTokenLen); err != nil {
		return err
	}
	if err := toolvalidation.OptionalTextFilter("http_path", input.HTTPPath, maxHTTPPathLength); err != nil {
		return err
	}
	if err := toolvalidation.OptionalSafeToken("reserved", input.Reserved, maxNetworkTokenLen); err != nil {
		return err
	}
	if err := toolvalidation.OptionalPort("port", input.Port); err != nil {
		return err
	}
	if err := toolvalidation.OptionalPort("to_port", input.ToPort); err != nil {
		return err
	}
	if input.Last < 0 || input.Last > maxFlowsLast {
		return fmt.Errorf("last must be between 0 and %d", maxFlowsLast)
	}
	return nil
}
