package tools

import (
	"context"
	"observability-hub/internal/mcp/providers"
)

// HubInput is the common input for hub-related tools.
type HubInput struct {
	Service string `json:"service,omitempty"` // service name for log query
	Since   string `json:"since,omitempty"`   // relative time for log query (e.g. 5m, 1h)
}

// InspectPlatformHandler handles executive summary of platform health.
type InspectPlatformHandler struct {
	inspectFn func(ctx context.Context) (map[string]interface{}, error)
}

func NewInspectPlatformHandler(fn func(ctx context.Context) (map[string]interface{}, error)) *InspectPlatformHandler {
	return &InspectPlatformHandler{inspectFn: fn}
}

func (h *InspectPlatformHandler) Execute(ctx context.Context, _ HubInput) (interface{}, error) {
	return h.inspectFn(ctx)
}

// InspectHostHandler handles physical resource inspection.
type InspectHostHandler struct {
	inspectFn func(ctx context.Context) (*providers.HostResource, error)
}

func NewInspectHostHandler(fn func(ctx context.Context) (*providers.HostResource, error)) *InspectHostHandler {
	return &InspectHostHandler{inspectFn: fn}
}

func (h *InspectHostHandler) Execute(ctx context.Context, _ HubInput) (interface{}, error) {
	return h.inspectFn(ctx)
}

// ListHostServicesHandler handles listing systemd units.
type ListHostServicesHandler struct {
	listFn func(ctx context.Context) ([]providers.ServiceStatus, error)
}

func NewListHostServicesHandler(fn func(ctx context.Context) ([]providers.ServiceStatus, error)) *ListHostServicesHandler {
	return &ListHostServicesHandler{listFn: fn}
}

func (h *ListHostServicesHandler) Execute(ctx context.Context, _ HubInput) (interface{}, error) {
	return h.listFn(ctx)
}

// QueryServiceLogsHandler handles journal log retrieval.
type QueryServiceLogsHandler struct {
	queryFn func(ctx context.Context, service string, since string) (string, error)
}

func NewQueryServiceLogsHandler(fn func(ctx context.Context, service string, since string) (string, error)) *QueryServiceLogsHandler {
	return &QueryServiceLogsHandler{queryFn: fn}
}

func (h *QueryServiceLogsHandler) Execute(ctx context.Context, input HubInput) (interface{}, error) {
	return h.queryFn(ctx, input.Service, input.Since)
}
