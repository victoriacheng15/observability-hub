package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesServiceList(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    ListOptions
		want    []Summary
	}{
		{
			name: "lists workload services",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 1),
				statefulSet("loki", "observability", created, replicas, 2),
				&batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "jobs", CreationTimestamp: created}},
			},
			want: []Summary{
				{Name: "backup", Namespace: "jobs", Kind: "CronJob", Ready: "active:0", Age: "2h"},
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/2", Age: "2h"},
				{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Ready: "2/2", Age: "2h"},
			},
		},
		{
			name: "filters services by namespace",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 1),
				deployment("payments-api", "payments", created, replicas, 2),
			},
			opts: ListOptions{Namespace: "payments"},
			want: []Summary{
				{Name: "payments-api", Namespace: "payments", Kind: "Deployment", Ready: "2/2", Age: "2h"},
			},
		},
		{
			name: "empty service list",
			want: []Summary{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)

			summaries, err := service.List(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			assertSummaries(t, summaries, tt.want)
		})
	}
}

func TestKubernetesServiceDescribe(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    DescribeOptions
		want    []Details
		wantErr string
	}{
		{
			name: "describes service by workload name",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 1),
			},
			opts: DescribeOptions{Name: "grafana"},
			want: []Details{
				{
					Summary:  Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/2", Age: "2h"},
					Images:   []string{"grafana/grafana:latest"},
					Labels:   map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
					Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
				},
			},
		},
		{
			name: "describes service by label",
			objects: []runtime.Object{
				deployment("grafana-web", "observability", created, replicas, 1),
			},
			opts: DescribeOptions{Name: "grafana"},
			want: []Details{
				{
					Summary:  Summary{Name: "grafana-web", Namespace: "observability", Kind: "Deployment", Ready: "1/2", Age: "2h"},
					Images:   []string{"grafana/grafana:latest"},
					Labels:   map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
					Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
				},
			},
		},
		{
			name: "describes namespace qualified service",
			objects: []runtime.Object{
				deployment("api", "payments", created, replicas, 2),
				deployment("api", "observability", created, replicas, 1),
			},
			opts: DescribeOptions{Name: "payments/api"},
			want: []Details{
				{
					Summary:  Summary{Name: "api", Namespace: "payments", Kind: "Deployment", Ready: "2/2", Age: "2h"},
					Images:   []string{"grafana/grafana:latest"},
					Labels:   map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
					Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
				},
			},
		},
		{
			name: "returns all duplicate service names across namespaces",
			objects: []runtime.Object{
				deployment("api", "payments", created, replicas, 2),
				deployment("api", "observability", created, replicas, 1),
			},
			opts: DescribeOptions{Name: "api"},
			want: []Details{
				{
					Summary:  Summary{Name: "api", Namespace: "observability", Kind: "Deployment", Ready: "1/2", Age: "2h"},
					Images:   []string{"grafana/grafana:latest"},
					Labels:   map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
					Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
				},
				{
					Summary:  Summary{Name: "api", Namespace: "payments", Kind: "Deployment", Ready: "2/2", Age: "2h"},
					Images:   []string{"grafana/grafana:latest"},
					Labels:   map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
					Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
				},
			},
		},
		{
			name:    "missing service",
			opts:    DescribeOptions{Name: "missing"},
			wantErr: "service not found: missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)

			details, err := service.Describe(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Describe() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Describe() error = %v", err)
			}

			assertDetails(t, details, tt.want)
		})
	}
}

func TestKubernetesServiceHealth(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    HealthOptions
		want    []Health
		wantErr string
	}{
		{
			name: "healthy service",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
				pod("grafana-0", "observability", true, 0),
				pod("grafana-1", "observability", true, 0),
				kubernetesService("grafana", "observability"),
				endpointSlice("grafana", "observability", true),
			},
			opts: HealthOptions{Name: "grafana"},
			want: []Health{
				{
					Summary:             Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "2/2", Age: "2h"},
					Status:              "healthy",
					ReadyPods:           2,
					PodCount:            2,
					ReadyServices:       1,
					ServiceCount:        1,
					WarningEventReasons: []string{},
				},
			},
		},
		{
			name: "restarted but currently ready service",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
				pod("grafana-0", "observability", true, 1),
				pod("grafana-1", "observability", true, 0),
				kubernetesService("grafana", "observability"),
				endpointSlice("grafana", "observability", true),
			},
			opts: HealthOptions{Name: "grafana"},
			want: []Health{
				{
					Summary:             Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "2/2", Age: "2h"},
					Status:              "healthy",
					ReadyPods:           2,
					PodCount:            2,
					Restarts:            1,
					ReadyServices:       1,
					ServiceCount:        1,
					WarningEventReasons: []string{},
				},
			},
		},
		{
			name: "degraded service",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 1),
				pod("grafana-0", "observability", true, 1),
				pod("grafana-1", "observability", false, 0),
				kubernetesService("grafana", "observability"),
				endpointSlice("grafana", "observability", false),
				warningEvent("grafana-1", "observability", "BackOff"),
			},
			opts: HealthOptions{Name: "grafana"},
			want: []Health{
				{
					Summary:             Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/2", Age: "2h"},
					Status:              "degraded",
					ReadyPods:           1,
					PodCount:            2,
					Restarts:            1,
					ReadyServices:       0,
					ServiceCount:        1,
					WarningEventCount:   1,
					WarningEventReasons: []string{"BackOff"},
				},
			},
		},
		{
			name:    "missing service",
			opts:    HealthOptions{Name: "missing"},
			wantErr: "service not found: missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)

			health, err := service.Health(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Health() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Health() error = %v", err)
			}

			assertHealth(t, health, tt.want)
		})
	}
}

func TestKubernetesServiceLogs(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    LogsOptions
		logs    map[string][]string
		want    []LogLine
		wantErr string
	}{
		{
			name: "reads logs for resolved service pods",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
				pod("grafana-0", "observability", true, 0),
				pod("grafana-1", "observability", true, 0),
			},
			opts: LogsOptions{Name: "grafana", Namespace: "observability", Container: "app", Tail: 10, Since: time.Minute},
			logs: map[string][]string{
				"observability/grafana-0/app": {"ready"},
				"observability/grafana-1/app": {"serving"},
			},
			want: []LogLine{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Pod: "grafana-0", Container: "app", Line: "ready"},
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Pod: "grafana-1", Container: "app", Line: "serving"},
			},
		},
		{
			name: "reads logs for pod name fallback",
			objects: []runtime.Object{
				pod("loki-0", "observability", true, 0),
			},
			opts: LogsOptions{Name: "loki-0", Tail: 5},
			logs: map[string][]string{
				"observability/loki-0": {"compactor ready"},
			},
			want: []LogLine{
				{Name: "loki-0", Namespace: "observability", Kind: "Pod", Pod: "loki-0", Line: "compactor ready"},
			},
		},
		{
			name: "reads all pod containers when container is not provided",
			objects: []runtime.Object{
				multiContainerPod("loki-0", "observability", "loki", "loki-sc-rules"),
			},
			opts: LogsOptions{Name: "loki-0", Tail: 5},
			logs: map[string][]string{
				"observability/loki-0/loki":          {"loki ready"},
				"observability/loki-0/loki-sc-rules": {"rules ready"},
			},
			want: []LogLine{
				{Name: "loki-0", Namespace: "observability", Kind: "Pod", Pod: "loki-0", Container: "loki", Line: "loki ready"},
				{Name: "loki-0", Namespace: "observability", Kind: "Pod", Pod: "loki-0", Container: "loki-sc-rules", Line: "rules ready"},
			},
		},
		{
			name: "reads logs for namespace qualified pod name fallback",
			objects: []runtime.Object{
				pod("loki-0", "observability", true, 0),
				pod("loki-0", "logging", true, 0),
			},
			opts: LogsOptions{Name: "logging/loki-0", Tail: 5},
			logs: map[string][]string{
				"logging/loki-0": {"querier ready"},
			},
			want: []LogLine{
				{Name: "loki-0", Namespace: "logging", Kind: "Pod", Pod: "loki-0", Line: "querier ready"},
			},
		},
		{
			name:    "missing service",
			opts:    LogsOptions{Name: "missing"},
			wantErr: "service not found: missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)
			service.podLogs = func(_ context.Context, pod corev1.Pod, container string, _ LogsOptions) ([]string, error) {
				if container == "" {
					return tt.logs[pod.Namespace+"/"+pod.Name], nil
				}
				return tt.logs[pod.Namespace+"/"+pod.Name+"/"+container], nil
			}

			logs, err := service.Logs(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Logs() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Logs() error = %v", err)
			}

			assertLogLines(t, logs, tt.want)
		})
	}
}

func TestKubernetesServiceEvents(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    EventsOptions
		want    []Event
		wantErr string
	}{
		{
			name: "returns recent workload and pod events",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
				pod("grafana-0", "observability", true, 0),
				event("grafana", "Deployment", "observability", corev1.EventTypeNormal, "Scaled", "scaled up", time.Date(2026, 4, 28, 11, 55, 0, 0, time.UTC)),
				event("grafana-0", "Pod", "observability", corev1.EventTypeWarning, "BackOff", "back-off restarting failed container", time.Date(2026, 4, 28, 11, 59, 0, 0, time.UTC)),
				event("other", "Pod", "observability", corev1.EventTypeWarning, "Failed", "ignored", time.Date(2026, 4, 28, 11, 59, 0, 0, time.UTC)),
			},
			opts: EventsOptions{Name: "grafana", Namespace: "observability", Tail: 10, Since: 10 * time.Minute},
			want: []Event{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Type: corev1.EventTypeWarning, Reason: "BackOff", Message: "back-off restarting failed container", Object: "Pod/grafana-0", Age: "1m", Time: time.Date(2026, 4, 28, 11, 59, 0, 0, time.UTC)},
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Type: corev1.EventTypeNormal, Reason: "Scaled", Message: "scaled up", Object: "Deployment/grafana", Age: "5m", Time: time.Date(2026, 4, 28, 11, 55, 0, 0, time.UTC)},
			},
		},
		{
			name: "limits events",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
				pod("grafana-0", "observability", true, 0),
				event("grafana", "Deployment", "observability", corev1.EventTypeNormal, "Old", "old", time.Date(2026, 4, 28, 11, 50, 0, 0, time.UTC)),
				event("grafana-0", "Pod", "observability", corev1.EventTypeWarning, "New", "new", time.Date(2026, 4, 28, 11, 59, 0, 0, time.UTC)),
			},
			opts: EventsOptions{Name: "grafana", Namespace: "observability", Tail: 1},
			want: []Event{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Type: corev1.EventTypeWarning, Reason: "New", Message: "new", Object: "Pod/grafana-0", Age: "1m", Time: time.Date(2026, 4, 28, 11, 59, 0, 0, time.UTC)},
			},
		},
		{
			name:    "missing service",
			opts:    EventsOptions{Name: "missing"},
			wantErr: "service not found: missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)

			events, err := service.Events(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Events() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Events() error = %v", err)
			}

			assertEvents(t, events, tt.want)
		})
	}
}

func TestKubernetesServiceMetrics(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name          string
		objects       []runtime.Object
		opts          MetricsOptions
		handler       http.Handler
		prometheusURL string
		want          []Metric
		wantErr       string
		wantQueries   int
	}{
		{
			name: "queries service metrics for resolved pods",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
				pod("grafana-0", "observability", true, 0),
				pod("grafana-1", "observability", true, 0),
			},
			opts:          MetricsOptions{Name: "grafana", Namespace: "observability", Window: 10 * time.Minute},
			prometheusURL: "http://prometheus",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/query" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				query := r.URL.Query().Get("query")
				value := "0"
				switch {
				case strings.Contains(query, "container_cpu_usage_seconds_total"):
					value = "0.12"
				case strings.Contains(query, "container_memory_working_set_bytes"):
					value = "1048576"
				case strings.Contains(query, "kube_pod_container_status_restarts_total"):
					value = "1"
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"success","data":{"result":[{"value":[1714291200,"` + value + `"]}]}}`))
			}),
			want: []Metric{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Signal: "cpu_cores", Value: "0.12"},
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Signal: "memory_bytes", Value: "1048576"},
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Signal: "restarts", Value: "1"},
			},
			wantQueries: 3,
		},
		{
			name:    "requires prometheus url",
			opts:    MetricsOptions{Name: "grafana"},
			wantErr: "PROMETHEUS_URL is required for service metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)
			var client *http.Client
			if tt.handler != nil {
				client = newInMemoryHTTPClient(tt.handler)
			}
			service.signals = newSignalClient(tt.prometheusURL, "http://tempo", client)

			metrics, err := service.Metrics(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Metrics() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Metrics() error = %v", err)
			}

			assertMetrics(t, metrics, tt.want)
		})
	}
}

func TestKubernetesServiceTraces(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)
	start := time.Date(2026, 4, 28, 11, 59, 0, 0, time.UTC).UnixNano()

	tests := []struct {
		name     string
		objects  []runtime.Object
		opts     TracesOptions
		handler  http.Handler
		tempoURL string
		want     []Trace
		wantErr  string
	}{
		{
			name: "searches traces for resolved service",
			objects: []runtime.Object{
				deployment("grafana", "observability", created, replicas, 2),
			},
			opts:     TracesOptions{Name: "grafana", Namespace: "observability", Hours: 2, Limit: 5},
			tempoURL: "http://tempo",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/search" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.URL.Query().Get("q") != `{resource.service.name="grafana"}` {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(fmt.Sprintf(`{"traces":[{"traceID":"abc123","rootServiceName":"grafana","startTimeUnixNano":"%d","durationMs":25}]}`, start)))
			}),
			want: []Trace{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", TraceID: "abc123", RootServiceName: "grafana", StartTime: "2026-04-28T11:59:00Z", Duration: "25ms"},
			},
		},
		{
			name:    "requires tempo url",
			opts:    TracesOptions{Name: "grafana"},
			wantErr: "TEMPO_URL is required for service traces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)
			var client *http.Client
			if tt.handler != nil {
				client = newInMemoryHTTPClient(tt.handler)
			}
			service.signals = newSignalClient("http://prometheus", tt.tempoURL, client)
			service.signals.now = func() time.Time {
				return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
			}

			traces, err := service.Traces(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Traces() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Traces() error = %v", err)
			}

			assertTraces(t, traces, tt.want)
		})
	}
}

func TestKubernetesServiceOwnership(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    OwnershipOptions
		want    []Ownership
		wantErr string
	}{
		{
			name: "reads ownership metadata from labels and annotations",
			objects: []runtime.Object{
				ownedDeployment("grafana", "observability", created, replicas),
			},
			opts: OwnershipOptions{Name: "grafana", Namespace: "observability"},
			want: []Ownership{
				{
					Summary:      Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "2/2", Age: "2h"},
					Owner:        "platform",
					Tier:         "frontend",
					Source:       "https://github.com/example/grafana",
					Docs:         []string{"platform.observability-hub.io/dashboard=https://grafana.example/d/grafana", "platform.observability-hub.io/runbook=docs/runbooks/grafana.md"},
					SourceObject: "Deployment observability/grafana",
				},
			},
		},
		{
			name: "returns unknown ownership when metadata is absent",
			objects: []runtime.Object{
				statefulSet("loki", "observability", created, replicas, 2),
			},
			opts: OwnershipOptions{Name: "loki", Namespace: "observability"},
			want: []Ownership{
				{
					Summary:      Summary{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Ready: "2/2", Age: "2h"},
					Owner:        "unknown",
					Tier:         "unknown",
					Source:       "unknown",
					Docs:         []string{"unknown"},
					SourceObject: "StatefulSet observability/loki",
				},
			},
		},
		{
			name:    "missing service",
			opts:    OwnershipOptions{Name: "missing"},
			wantErr: "service not found: missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(tt.objects...)

			ownership, err := service.Ownership(context.Background(), tt.opts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Ownership() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Ownership() error = %v", err)
			}

			assertOwnership(t, ownership, tt.want)
		})
	}
}

func deployment(name string, namespace string, created metav1.Time, replicas int32, ready int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: created,
			Labels:            map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/name": "grafana",
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "grafana/grafana:latest"}},
				},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: ready},
	}
}

func ownedDeployment(name string, namespace string, created metav1.Time, replicas int32) *appsv1.Deployment {
	deployment := deployment(name, namespace, created, replicas, replicas)
	deployment.Labels["app.kubernetes.io/owner"] = "platform"
	deployment.Labels["tier"] = "frontend"
	deployment.Annotations = map[string]string{
		"platform.observability-hub.io/repo":      "https://github.com/example/grafana",
		"platform.observability-hub.io/runbook":   "docs/runbooks/grafana.md",
		"platform.observability-hub.io/dashboard": "https://grafana.example/d/grafana",
	}
	return deployment
}

func statefulSet(name string, namespace string, created metav1.Time, replicas int32, ready int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, CreationTimestamp: created},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/name": name,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "grafana/loki:latest"}},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{ReadyReplicas: ready},
	}
}

func pod(name string, namespace string, ready bool, restarts int32) *corev1.Pod {
	conditionStatus := corev1.ConditionFalse
	if ready {
		conditionStatus = corev1.ConditionTrue
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app.kubernetes.io/name": "grafana"},
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: conditionStatus},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", RestartCount: restarts},
			},
		},
	}
}

func multiContainerPod(name string, namespace string, containers ...string) *corev1.Pod {
	pod := pod(name, namespace, true, 0)
	pod.Spec.Containers = make([]corev1.Container, 0, len(containers))
	for _, container := range containers {
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: container})
	}
	return pod
}

func kubernetesService(name string, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
		},
	}
}

func endpointSlice(name string, namespace string, ready bool) *discoveryv1.EndpointSlice {
	return &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				discoveryv1.LabelServiceName: name,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"10.0.0.1"},
				Conditions: discoveryv1.EndpointConditions{
					Ready: &ready,
				},
			},
		},
	}
}

func warningEvent(name string, namespace string, reason string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: name + "." + reason, Namespace: namespace},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      name,
			Namespace: namespace,
		},
		Type:   corev1.EventTypeWarning,
		Reason: reason,
	}
}

func event(name string, kind string, namespace string, eventType string, reason string, message string, lastSeen time.Time) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: name + "." + reason, Namespace: namespace},
		InvolvedObject: corev1.ObjectReference{
			Kind:      kind,
			Name:      name,
			Namespace: namespace,
		},
		Type:          eventType,
		Reason:        reason,
		Message:       message,
		LastTimestamp: metav1.NewTime(lastSeen),
	}
}

func assertSummaries(t *testing.T, got []Summary, want []Summary) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(summaries) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("summaries[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertHealth(t *testing.T, got []Health, want []Health) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(health) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Summary != want[i].Summary {
			t.Fatalf("health[%d].Summary = %#v, want %#v", i, got[i].Summary, want[i].Summary)
		}
		if got[i].Status != want[i].Status ||
			got[i].ReadyPods != want[i].ReadyPods ||
			got[i].PodCount != want[i].PodCount ||
			got[i].Restarts != want[i].Restarts ||
			got[i].ReadyServices != want[i].ReadyServices ||
			got[i].ServiceCount != want[i].ServiceCount ||
			got[i].WarningEventCount != want[i].WarningEventCount {
			t.Fatalf("health[%d] = %#v, want %#v", i, got[i], want[i])
		}
		assertStringSlice(t, got[i].WarningEventReasons, want[i].WarningEventReasons)
	}
}

func assertLogLines(t *testing.T, got []LogLine, want []LogLine) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(logs) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("logs[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertEvents(t *testing.T, got []Event, want []Event) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(events) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertMetrics(t *testing.T, got []Metric, want []Metric) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(metrics) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("metrics[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertTraces(t *testing.T, got []Trace, want []Trace) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(traces) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("traces[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertOwnership(t *testing.T, got []Ownership, want []Ownership) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(ownership) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Summary != want[i].Summary ||
			got[i].Owner != want[i].Owner ||
			got[i].Tier != want[i].Tier ||
			got[i].Source != want[i].Source ||
			got[i].SourceObject != want[i].SourceObject {
			t.Fatalf("ownership[%d] = %#v, want %#v", i, got[i], want[i])
		}
		assertStringSlice(t, got[i].Docs, want[i].Docs)
	}
}

func assertDetails(t *testing.T, got []Details, want []Details) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(details) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Summary != want[i].Summary {
			t.Fatalf("details[%d].Summary = %#v, want %#v", i, got[i].Summary, want[i].Summary)
		}
		assertStringSlice(t, got[i].Images, want[i].Images)
		assertStringMap(t, got[i].Labels, want[i].Labels)
		assertStringMap(t, got[i].Selector, want[i].Selector)
	}
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(slice) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func assertStringMap(t *testing.T, got map[string]string, want map[string]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(map) = %d, want %d: %#v", len(got), len(want), got)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("map[%q] = %q, want %q", key, got[key], value)
		}
	}
}

func newTestService(objects ...runtime.Object) *KubernetesService {
	service := NewKubernetesServiceWithClientset(fake.NewSimpleClientset(objects...))
	service.now = func() time.Time {
		return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	}
	return service
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newInMemoryHTTPClient(handler http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			cloned := req.Clone(req.Context())
			cloned.RequestURI = cloned.URL.RequestURI()
			handler.ServeHTTP(recorder, cloned)
			response := recorder.Result()
			response.Request = req
			return response, nil
		}),
	}
}
