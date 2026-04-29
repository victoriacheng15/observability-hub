package catalog

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

func TestKubernetesCatalogList(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)
	completions := int32(1)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    ListOptions
		want    []Entry
	}{
		{
			name: "mixed kubernetes catalog entries",
			objects: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability", CreationTimestamp: created},
					Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability", CreationTimestamp: created},
					Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
					Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: "loki", Namespace: "observability", CreationTimestamp: created},
					Spec:       appsv1.StatefulSetSpec{Replicas: &replicas},
					Status:     appsv1.StatefulSetStatus{ReadyReplicas: 2},
				},
				&batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{Name: "analytics", Namespace: "workers", CreationTimestamp: created},
					Spec:       batchv1.JobSpec{Completions: &completions},
					Status:     batchv1.JobStatus{Succeeded: 1},
				},
			},
			want: []Entry{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Status: "1/2", Age: "2h"},
				{Name: "grafana", Namespace: "observability", Kind: "Service", Status: "ClusterIP", Age: "2h"},
				{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Status: "2/2", Age: "2h"},
				{Name: "analytics", Namespace: "workers", Kind: "Job", Status: "1/1", Age: "2h"},
			},
		},
		{
			name: "filters by namespace",
			objects: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability", CreationTimestamp: created},
					Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "payments-api", Namespace: "payments", CreationTimestamp: created},
					Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
					Status:     appsv1.DeploymentStatus{ReadyReplicas: 2},
				},
			},
			opts: ListOptions{Namespace: "payments"},
			want: []Entry{
				{Name: "payments-api", Namespace: "payments", Kind: "Deployment", Status: "2/2", Age: "2h"},
			},
		},
		{
			name: "empty catalog",
			want: []Entry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newTestCatalog(tt.objects...)

			entries, err := catalog.List(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			assertEntries(t, entries, tt.want)
		})
	}
}

func TestKubernetesCatalogValidate(t *testing.T) {
	tests := []struct {
		name    string
		objects []runtime.Object
		opts    ListOptions
		want    Validation
	}{
		{
			name: "counts resources across namespaces",
			objects: []runtime.Object{
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability"}},
				&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability"}},
				&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "analytics", Namespace: "workers"}},
			},
			want: Validation{
				ResourceCount:  3,
				NamespaceCount: 2,
				KindCounts: map[string]int{
					"Deployment": 1,
					"Job":        1,
					"Service":    1,
				},
			},
		},
		{
			name: "counts filtered namespace",
			objects: []runtime.Object{
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability"}},
				&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "payments-api", Namespace: "payments"}},
				&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "payments-reconcile", Namespace: "payments"}},
			},
			opts: ListOptions{Namespace: "payments"},
			want: Validation{
				ResourceCount:  2,
				NamespaceCount: 1,
				KindCounts: map[string]int{
					"Deployment": 1,
					"Job":        1,
				},
			},
		},
		{
			name: "empty catalog",
			want: Validation{KindCounts: map[string]int{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newTestCatalog(tt.objects...)

			validation, err := catalog.Validate(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if validation.ResourceCount != tt.want.ResourceCount {
				t.Fatalf("ResourceCount = %d, want %d", validation.ResourceCount, tt.want.ResourceCount)
			}
			if validation.NamespaceCount != tt.want.NamespaceCount {
				t.Fatalf("NamespaceCount = %d, want %d", validation.NamespaceCount, tt.want.NamespaceCount)
			}
			if len(validation.KindCounts) != len(tt.want.KindCounts) {
				t.Fatalf("Validate() = %#v, want %#v", validation, tt.want)
			}
			for kind, count := range tt.want.KindCounts {
				if validation.KindCounts[kind] != count {
					t.Fatalf("KindCounts[%q] = %d, want %d", kind, validation.KindCounts[kind], count)
				}
			}
		})
	}
}

func assertEntries(t *testing.T, got []Entry, want []Entry) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(entries) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entries[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func newTestCatalog(objects ...runtime.Object) *KubernetesCatalog {
	catalog := NewKubernetesCatalogWithClientset(fake.NewSimpleClientset(objects...))
	catalog.now = func() time.Time {
		return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	}
	return catalog
}
