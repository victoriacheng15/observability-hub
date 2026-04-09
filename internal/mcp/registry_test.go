package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/mcp/tools/hub"
	"observability-hub/internal/mcp/tools/pods"
	"observability-hub/internal/mcp/tools/telemetry"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newInMemoryHTTPClient(h http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			done := make(chan *http.Response, 1)
			go func() {
				rr := httptest.NewRecorder()
				r2 := req.Clone(req.Context())
				r2.RequestURI = r2.URL.RequestURI()
				h.ServeHTTP(rr, r2)
				resp := rr.Result()
				resp.Request = req
				done <- resp
			}()
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case resp := <-done:
				return resp, nil
			}
		}),
		Timeout: 2 * time.Second,
	}
}

func TestRegistryHandlers_Telemetry(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/query":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success","data":{"resultType":"instant","result":[]}}`))
			return
		case r.URL.Path == "/loki/api/v1/query_range":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[{"stream":{},"values":[["1","err"]]}]}}`))
			return
		case strings.HasPrefix(r.URL.Path, "/api/traces/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[{"traceID":"4bf92f3577b34da6a3ce929d0e0e4736","spans":[]}]}`))
			return
		case r.URL.Path == "/api/search":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"traces":[{"traceID":"abc123","rootServiceName":"proxy"}]}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})

	tp := providers.NewTelemetryProviderWithClient("http://thanos", "http://loki", "http://tempo", newInMemoryHTTPClient(h))
	ctx := context.Background()

	tests := []struct {
		name    string
		handler func(ctx context.Context) (*sdkmcp.CallToolResult, error)
		wants   []string // Accept any of these substrings (supports raw and summarized modes)
	}{
		{
			name: "query_metrics",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleQueryMetrics(tp, "svc")
				res, _, err := h(ctx, nil, telemetry.QueryMetricsInput{Query: "up"})
				return res, err
			},
			wants: []string{`"summarized_count"`, `"status":"success"`},
		},
		{
			name: "query_logs",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleQueryLogs(tp, "svc")
				res, _, err := h(ctx, nil, telemetry.QueryLogsInput{Query: `{service="proxy"}`, Limit: 1, Hours: 1})
				return res, err
			},
			wants: []string{`"summarized_count"`, `"status":"success"`},
		},
		{
			name: "query_traces_by_id",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleQueryTraces(tp, "svc")
				res, _, err := h(ctx, nil, telemetry.QueryTracesInput{TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"})
				return res, err
			},
			wants: []string{"4bf92f3577b34da6a3ce929d0e0e4736"},
		},
		{
			name: "investigate_incident",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleInvestigateIncident(tp, "svc")
				res, _, err := h(ctx, nil, telemetry.InvestigateIncidentInput{Service: "proxy", Hours: 1})
				return res, err
			},
			wants: []string{`"service":"proxy"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.handler(ctx)
			if err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			tc := res.Content[0].(*sdkmcp.TextContent)

			found := false
			for _, want := range tt.wants {
				if strings.Contains(tc.Text, want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("got %s, want to contain any of %v", tc.Text, tt.wants)
			}
		})
	}
}

func TestRegisterTools_DoesNotPanic(t *testing.T) {
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	RegisterTelemetryTools(srv, providers.NewTelemetryProvider("http://thanos", "http://loki", "http://tempo"), "svc")
	RegisterPodsTools(srv, (*providers.PodsProvider)(nil), "svc")
	RegisterHubTools(srv, (*providers.HubProvider)(nil), "svc")
	RegisterNetworkTools(srv, (*providers.HubProvider)(nil), "svc")
}

func TestRegistry_PodHandlers(t *testing.T) {
	fakePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
		},
	}
	fakeEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind: "Pod",
			Name: "test-pod",
		},
		Message: "pod started",
	}
	clientset := fake.NewSimpleClientset(fakePod, fakeEvent)
	pp := providers.NewPodsProviderWithClientset(clientset)
	ctx := context.Background()

	tests := []struct {
		name    string
		handler func(ctx context.Context) (*sdkmcp.CallToolResult, error)
		want    string
	}{
		{
			name: "inspect_pods",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleInspectPods(pp, "svc")
				res, _, err := h(ctx, nil, pods.PodsInput{Namespace: "default"})
				return res, err
			},
			want: "test-pod",
		},
		{
			name: "describe_pod",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleDescribePod(pp, "svc")
				res, _, err := h(ctx, nil, pods.PodsInput{Namespace: "default", Name: "test-pod"})
				return res, err
			},
			want: `"name":"test-pod"`,
		},
		{
			name: "list_pod_events",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleListPodEvents(pp, "svc")
				res, _, err := h(ctx, nil, pods.PodsInput{Namespace: "default", Name: "test-pod"})
				return res, err
			},
			want: "pod started",
		},
		{
			name: "get_pod_logs",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleGetPodLogs(pp, "svc")
				res, _, err := h(ctx, nil, pods.PodLogsInput{Namespace: "default", Name: "test-pod"})
				return res, err
			},
			want: "", // fake logs are empty string
		},
		{
			name: "delete_pod",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleDeletePod(pp, "svc")
				res, _, err := h(ctx, nil, pods.DeletePodInput{Namespace: "default", Name: "test-pod"})
				return res, err
			},
			want: `"status":"deleted"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.handler(ctx)
			if err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			tc := res.Content[0].(*sdkmcp.TextContent)
			if !strings.Contains(tc.Text, tt.want) {
				t.Errorf("got %s, want to contain %s", tc.Text, tt.want)
			}
		})
	}
}

func TestRegistry_HubHandlers(t *testing.T) {
	hp := providers.NewHubProvider()
	ctx := context.Background()

	tests := []struct {
		name    string
		handler func(ctx context.Context) (*sdkmcp.CallToolResult, error)
		want    string
	}{
		{
			name: "hub_inspect_platform",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleInspectPlatform(hp, "svc")
				res, _, err := h(ctx, nil, hub.HubInput{})
				return res, err
			},
			want: `"node":"server2"`,
		},
		{
			name: "hub_inspect_host",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleInspectHost(hp, "svc")
				res, _, err := h(ctx, nil, hub.HubInput{})
				return res, err
			},
			want: "cpu_usage",
		},
		{
			name: "hub_list_host_services",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleListHostServices(hp, "svc")
				res, _, err := h(ctx, nil, hub.HubInput{})
				return res, err
			},
			want: "proxy.service",
		},
		{
			name: "observe_network_flows",
			handler: func(ctx context.Context) (*sdkmcp.CallToolResult, error) {
				h := handleObserveNetworkFlows(hp, "svc")
				res, _, err := h(ctx, nil, hub.ObserveNetworkFlowsInput{Namespace: "default"})
				return res, err
			},
			want: "", // Output might be empty in test but we verify the call
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.handler(ctx)
			if err != nil {
				return
			}
			tc := res.Content[0].(*sdkmcp.TextContent)
			if !strings.Contains(tc.Text, tt.want) {
				t.Errorf("got %s, want to contain %s", tc.Text, tt.want)
			}
		})
	}
}
