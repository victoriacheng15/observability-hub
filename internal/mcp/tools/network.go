package tools

import (
	"context"
)

// ObserveNetworkFlowsInput is the input for the observe_network_flows tool.
type ObserveNetworkFlowsInput struct {
	Namespace string `json:"namespace,omitempty"` // filter by namespace
	Pod       string `json:"pod,omitempty"`       // filter by pod
	Reserved  string `json:"reserved,omitempty"`  // filter by reserved entity (e.g. "host", "world")
	Last      int    `json:"last,omitempty"`      // number of recent flows (default 20, max 100)
}

// ObserveNetworkFlowsHandler handles real-time network flow observation via Hubble.
type ObserveNetworkFlowsHandler struct {
	queryFn func(ctx context.Context, namespace, pod, reserved string, last int) (string, error)
}

func NewObserveNetworkFlowsHandler(fn func(ctx context.Context, namespace, pod, reserved string, last int) (string, error)) *ObserveNetworkFlowsHandler {
	return &ObserveNetworkFlowsHandler{queryFn: fn}
}

func (h *ObserveNetworkFlowsHandler) Execute(ctx context.Context, input ObserveNetworkFlowsInput) (interface{}, error) {
	return h.queryFn(ctx, input.Namespace, input.Pod, input.Reserved, input.Last)
}
