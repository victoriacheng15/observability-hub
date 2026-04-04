package providers

import (
	"context"
	"errors"
	"reflect"
	"strings"
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
				"proxy.service":          "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"openbao.service":        "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"tailscale-gate.service": "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
			},
			expectedStatus: []ServiceStatus{
				{Name: "proxy.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "openbao.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
				{Name: "tailscale-gate.service", Active: "active", Sub: "running", Since: "Wed 2026-03-11"},
			},
		},
		{
			name: "Service Inactive",
			mockOutput: map[string]string{
				"proxy.service":          "ActiveState=inactive\nSubState=dead\nActiveEnterTimestamp=",
				"openbao.service":        "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
				"tailscale-gate.service": "ActiveState=active\nSubState=running\nActiveEnterTimestamp=Wed 2026-03-11",
			},
			expectedStatus: []ServiceStatus{
				{Name: "proxy.service", Active: "inactive", Sub: "dead", Since: ""},
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

func TestHubProvider_QueryHubbleFlows(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		pod        string
		fromPod    string
		toPod      string
		protocol   string
		verdict    string
		httpStatus string
		httpMethod string
		httpPath   string
		reserved   string
		port       int
		toPort     int
		last       int
		mockOutput string
		mockErr    error
		wantErr    bool
		wantArgs   []string
	}{
		{
			name:       "Basic Filters",
			namespace:  "default",
			pod:        "proxy",
			last:       10,
			mockOutput: `{"flow":{}}`,
			wantErr:    false,
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "10", "--output", "json", "--namespace", "default", "--pod", "proxy"},
		},
		{
			name:       "Directional Pod Filters",
			fromPod:    "default/frontend",
			toPod:      "default/backend",
			mockOutput: `{"flow":{}}`,
			wantErr:    false,
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "20", "--output", "json", "--from-pod", "default/frontend", "--to-pod", "default/backend"},
		},
		{
			name:       "L4/L7 and Verdict Filters",
			protocol:   "tcp",
			port:       80,
			toPort:     8080,
			verdict:    "DROPPED",
			mockOutput: `{"flow":{}}`,
			wantErr:    false,
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "20", "--output", "json", "--protocol", "tcp", "--port", "80", "--to-port", "8080", "--verdict", "DROPPED"},
		},
		{
			name:       "Reserved Entity Filter",
			reserved:   "host",
			last:       5,
			mockOutput: "Defaulted container\n" + `{"flow":{}}`,
			wantErr:    false,
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "5", "--output", "json", "--label", "reserved:host"},
		},
		{
			name:    "Command Failure",
			mockErr: errors.New("kubectl exec error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommandRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					if name != "kubectl" {
						t.Errorf("expected kubectl command, got %s", name)
					}
					if tt.wantArgs != nil {
						if !reflect.DeepEqual(arg, tt.wantArgs) {
							t.Errorf("got args %v, want %v", arg, tt.wantArgs)
						}
					}
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			p := &HubProvider{runner: mock}

			got, err := p.QueryHubbleFlows(context.Background(), tt.namespace, tt.pod, tt.fromPod, tt.toPod, tt.protocol, tt.verdict, tt.httpStatus, tt.httpMethod, tt.httpPath, tt.reserved, tt.port, tt.toPort, tt.last)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryHubbleFlows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if strings.Contains(got, "Defaulted container") {
					t.Errorf("output contains noise: %q", got)
				}
				if got != `{"flow":{}}` {
					t.Errorf("got %q, want %q", got, `{"flow":{}}`)
				}
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
			wantSvc:    "3/3",
		},
		{
			name:       "K3s Unreachable",
			kubectlErr: errors.New("connection refused"),
			mockSvcOut: "ActiveState=inactive\nSubState=dead\nActiveEnterTimestamp=",
			wantK3s:    "unreachable",
			wantSvc:    "3/3",
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
				targetServices: []string{"s1", "s2", "s3"},
			}

			got, err := p.InspectPlatform(context.Background())
			if err != nil {
				t.Fatalf("InspectPlatform() error = %v", err)
			}
			if got["k3s_status"] != tt.wantK3s {
				t.Errorf("k3s_status = %v, want %v", got["k3s_status"], tt.wantK3s)
			}
			if got["host_services_healthy"] != tt.wantSvc {
				t.Errorf("host_services_healthy = %v, want %v", got["host_services_healthy"], tt.wantSvc)
			}
		})
	}
}
