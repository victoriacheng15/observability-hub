package providers

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// MockCommandRunner satisfies the CommandRunner interface for testing.
type MockCommandRunner struct {
	RunFn func(ctx context.Context, name string, arg ...string) ([]byte, error)
}

func (m *MockCommandRunner) Run(ctx context.Context, name string, arg ...string) ([]byte, error) {
	if m.RunFn != nil {
		return m.RunFn(ctx, name, arg...)
	}
	return nil, nil
}

func TestHubProvider_ListHostServices(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     map[string]string
		mockErr        error
		expectedStatus []ServiceStatus
	}{
		{
			name: "All Services Active",
			mockOutput: map[string]string{
				"ingestion.service":      "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"proxy.service":          "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"openbao.service":        "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"tailscale-gate.service": "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
			},
			expectedStatus: []ServiceStatus{
				{Name: "ingestion.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "proxy.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "openbao.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "tailscale-gate.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
			},
		},
		{
			name: "Service Inactive",
			mockOutput: map[string]string{
				"ingestion.service":      "ActiveState=inactive\nSubState=dead\nActiveEnterTimestamp=",
				"proxy.service":          "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"openbao.service":        "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"tailscale-gate.service": "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
			},
			expectedStatus: []ServiceStatus{
				{Name: "ingestion.service", Active: "inactive", Sub: "dead", Since: ""},
				{Name: "proxy.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "openbao.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "tailscale-gate.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommandRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					if name == "systemctl" && len(arg) > 2 {
						// systemctl show <svc> --property=...
						svc := arg[1]
						if out, ok := tt.mockOutput[svc]; ok {
							return []byte(out), nil
						}
					}
					return nil, tt.mockErr
				},
			}
			p := &HubProvider{
				runner: mock,
				targetServices: []string{
					"ingestion.service",
					"proxy.service",
					"openbao.service",
					"tailscale-gate.service",
				},
			}

			got, err := p.ListHostServices(context.Background())
			if err != nil {
				t.Fatalf("ListHostServices() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.expectedStatus) {
				t.Errorf("got %v, want %v", got, tt.expectedStatus)
			}
		})
	}
}

func TestHubProvider_QueryServiceLogs(t *testing.T) {
	tests := []struct {
		name       string
		service    string
		since      string
		mockOutput string
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "Successful Log Retrieval",
			service:    "proxy.service",
			since:      "5m",
			mockOutput: "Mar 11 14:00:00 proxy logs...",
			wantErr:    false,
		},
		{
			name:    "Command Failure",
			service: "proxy.service",
			mockErr: errors.New("journalctl error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommandRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			p := &HubProvider{runner: mock}

			got, err := p.QueryServiceLogs(context.Background(), tt.service, tt.since)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryServiceLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.mockOutput {
				t.Errorf("got %q, want %q", got, tt.mockOutput)
			}
		})
	}
}

func TestHubProvider_InspectHost(t *testing.T) {
	tests := []struct {
		name      string
		uptimeOut string
		freeOut   string
		dfOut     string
		expected  *HostResource
	}{
		{
			name:      "Successful Inspection",
			uptimeOut: "14:30:00 up 1 day, load average: 0.05, 0.10, 0.15",
			freeOut:   "              total        used        free\nMem:           31Gi        4.5Gi        20Gi",
			dfOut:     "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       100G   40G   60G  40% /",
			expected: &HostResource{
				LoadAverage: "14:30:00 up 1 day, load average: 0.05, 0.10, 0.15",
				MemoryTotal: "31Gi",
				MemoryUsed:  "4.5Gi",
				DiskUsage:   "40%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommandRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					switch name {
					case "uptime":
						return []byte(tt.uptimeOut), nil
					case "free":
						return []byte(tt.freeOut), nil
					case "df":
						return []byte(tt.dfOut), nil
					}
					return nil, nil
				},
			}
			p := &HubProvider{runner: mock}

			got, err := p.InspectHost(context.Background())
			if err != nil {
				t.Fatalf("InspectHost() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestHubProvider_InspectPlatform(t *testing.T) {
	tests := []struct {
		name       string
		kubectlErr error
		mockSvcOut string
		wantK3s    string
		wantSvc    string
	}{
		{
			name:       "K3s Healthy",
			kubectlErr: nil,
			mockSvcOut: "ActiveState=active\nSubState=running\nActiveEnterTimestamp=now",
			wantK3s:    "healthy",
			wantSvc:    "4/4",
		},
		{
			name:       "K3s Unreachable",
			kubectlErr: errors.New("connection refused"),
			mockSvcOut: "ActiveState=inactive\nSubState=dead\nActiveEnterTimestamp=",
			wantK3s:    "unreachable",
			wantSvc:    "0/4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommandRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					if name == "kubectl" {
						return nil, tt.kubectlErr
					}
					if name == "systemctl" {
						return []byte(tt.mockSvcOut), nil
					}
					return nil, nil
				},
			}
			p := &HubProvider{
				runner:         mock,
				targetServices: []string{"s1", "s2", "s3", "s4"},
			}

			got, err := p.InspectPlatform(context.Background())
			if err != nil {
				t.Fatalf("InspectPlatform() error = %v", err)
			}
			if got["k3s_status"] != tt.wantK3s {
				t.Errorf("k3s_status = %v, want %v", got["k3s_status"], tt.wantK3s)
			}
			if got["host_services_running"] != tt.wantSvc {
				t.Errorf("host_services_running = %v, want %v", got["host_services_running"], tt.wantSvc)
			}
		})
	}
}
