package hub

import (
	"context"
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
	// queryFn is decoupled from the Input struct to prevent import cycles.
	queryFn func(ctx context.Context, namespace, pod, fromPod, toPod, protocol, verdict, httpStatus, httpMethod, httpPath, reserved string, port, toPort, last int) (string, error)
}

func NewObserveNetworkFlowsHandler(fn func(ctx context.Context, namespace, pod, fromPod, toPod, protocol, verdict, httpStatus, httpMethod, httpPath, reserved string, port, toPort, last int) (string, error)) *ObserveNetworkFlowsHandler {
	return &ObserveNetworkFlowsHandler{queryFn: fn}
}

func (h *ObserveNetworkFlowsHandler) Execute(ctx context.Context, input ObserveNetworkFlowsInput) (interface{}, error) {
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
