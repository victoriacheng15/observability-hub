package service

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
