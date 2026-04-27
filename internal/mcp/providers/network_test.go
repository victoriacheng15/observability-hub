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

func TestNetworkProvider_QueryHubbleFlows(t *testing.T) {
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
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "10", "--output", "json", "--namespace", "default", "--pod", "proxy"},
		},
		{
			name:       "Directional Pod Filters",
			fromPod:    "default/frontend",
			toPod:      "default/backend",
			mockOutput: `{"flow":{}}`,
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "20", "--output", "json", "--from-pod", "default/frontend", "--to-pod", "default/backend"},
		},
		{
			name:       "L4/L7 and Verdict Filters",
			protocol:   "tcp",
			port:       80,
			toPort:     8080,
			verdict:    "DROPPED",
			mockOutput: `{"flow":{}}`,
			wantArgs:   []string{"-n", "kube-system", "exec", "ds/cilium", "--", "hubble", "--server", "unix:///var/run/cilium/hubble.sock", "observe", "--last", "20", "--output", "json", "--protocol", "tcp", "--port", "80", "--to-port", "8080", "--verdict", "DROPPED"},
		},
		{
			name:       "Reserved Entity Filter",
			reserved:   "host",
			last:       5,
			mockOutput: "Defaulted container\n" + `{"flow":{}}`,
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
					if tt.wantArgs != nil && !reflect.DeepEqual(arg, tt.wantArgs) {
						t.Errorf("got args %v, want %v", arg, tt.wantArgs)
					}
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			p := &NetworkProvider{runner: mock}

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
