package mcp

import (
	"context"
	"sync"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// CapabilitySnapshot reports the MCP domains and tools registered at startup.
type CapabilitySnapshot struct {
	ServerVersion string             `json:"server_version"`
	ToolCount     int                `json:"tool_count"`
	Domains       []CapabilityDomain `json:"domains"`
}

// CapabilityDomain describes one logical MCP domain.
type CapabilityDomain struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Tools  []string `json:"tools,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

// CapabilityRegistry tracks startup capability state for logging and diagnostics.
type CapabilityRegistry struct {
	mu            sync.RWMutex
	serverVersion string
	domains       []CapabilityDomain
}

// NewCapabilityRegistry creates a registry for the current MCP process.
func NewCapabilityRegistry(serverVersion string) *CapabilityRegistry {
	return &CapabilityRegistry{serverVersion: serverVersion}
}

// AddAvailable records a registered tool domain.
func (r *CapabilityRegistry) AddAvailable(name string, tools []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.domains = append(r.domains, CapabilityDomain{
		Name:   name,
		Status: "available",
		Tools:  append([]string(nil), tools...),
	})
}

// AddSkipped records a domain that was not registered.
func (r *CapabilityRegistry) AddSkipped(name, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.domains = append(r.domains, CapabilityDomain{
		Name:   name,
		Status: "skipped",
		Reason: reason,
	})
}

// Snapshot returns a copy of the current startup capability state.
func (r *CapabilityRegistry) Snapshot() CapabilitySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	domains := make([]CapabilityDomain, 0, len(r.domains))
	toolCount := 0
	for _, domain := range r.domains {
		copied := domain
		copied.Tools = append([]string(nil), domain.Tools...)
		toolCount += len(copied.Tools)
		domains = append(domains, copied)
	}

	return CapabilitySnapshot{
		ServerVersion: r.serverVersion,
		ToolCount:     toolCount,
		Domains:       domains,
	}
}

// RegisterCapabilityTools registers diagnostic MCP tools.
func RegisterCapabilityTools(server *sdkmcp.Server, registry *CapabilityRegistry, serviceName string) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "mcp_capabilities",
		Description: "Report MCP domains, registered tools, and skipped providers for this gateway process",
	}, handleMCPCapabilities(registry, serviceName))
}

func handleMCPCapabilities(registry *CapabilityRegistry, serviceName string) sdkmcp.ToolHandlerFor[struct{}, any] {
	return NewJSONToolHandler("mcp_capabilities", serviceName, func(ctx context.Context, input struct{}) (interface{}, error) {
		return registry.Snapshot(), nil
	})
}
