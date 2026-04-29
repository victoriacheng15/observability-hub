package service

import (
	"context"
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
