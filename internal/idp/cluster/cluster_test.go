package cluster

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

func TestKubernetesClusterStatus(t *testing.T) {
	tests := []struct {
		name    string
		objects []runtime.Object
		want    Status
	}{
		{
			name: "counts ready nodes namespaces and workloads",
			objects: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "server-a"},
					Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					}},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "server-b"},
					Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					}},
				},
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "observability"}},
				&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability"}},
				&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "analytics", Namespace: "default"}},
			},
			want: Status{
				ReadyNodes:     1,
				NodeCount:      2,
				NamespaceCount: 2,
				WorkloadCount:  2,
			},
		},
		{
			name: "empty cluster",
			want: Status{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := newTestCluster(tt.objects...)

			status, err := cluster.Status(context.Background())
			if err != nil {
				t.Fatalf("Status() error = %v", err)
			}

			if status != tt.want {
				t.Fatalf("Status() = %#v, want %#v", status, tt.want)
			}
		})
	}
}

func TestKubernetesClusterNamespaces(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))

	tests := []struct {
		name    string
		objects []runtime.Object
		want    []Namespace
	}{
		{
			name: "lists namespaces sorted by name",
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "observability", CreationTimestamp: created},
					Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "payments", CreationTimestamp: created},
					Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
				},
			},
			want: []Namespace{
				{Name: "observability", Phase: "Active", Age: "2h"},
				{Name: "payments", Phase: "Terminating", Age: "2h"},
			},
		},
		{
			name: "empty namespaces",
			want: []Namespace{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := newTestCluster(tt.objects...)

			namespaces, err := cluster.Namespaces(context.Background())
			if err != nil {
				t.Fatalf("Namespaces() error = %v", err)
			}

			assertNamespaces(t, namespaces, tt.want)
		})
	}
}

func TestKubernetesClusterWorkloads(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	replicas := int32(2)
	completions := int32(1)

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    WorkloadOptions
		want    []Workload
	}{
		{
			name: "lists workloads sorted by namespace kind and name",
			objects: []runtime.Object{
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
			want: []Workload{
				{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/2", Age: "2h"},
				{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Ready: "2/2", Age: "2h"},
				{Name: "analytics", Namespace: "workers", Kind: "Job", Ready: "1/1", Age: "2h"},
			},
		},
		{
			name: "filters workloads by namespace",
			objects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability", CreationTimestamp: created},
					Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
					Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "payments-api", Namespace: "payments", CreationTimestamp: created},
					Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
					Status:     appsv1.DeploymentStatus{ReadyReplicas: 2},
				},
			},
			opts: WorkloadOptions{Namespace: "payments"},
			want: []Workload{
				{Name: "payments-api", Namespace: "payments", Kind: "Deployment", Ready: "2/2", Age: "2h"},
			},
		},
		{
			name: "empty workloads",
			want: []Workload{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := newTestCluster(tt.objects...)

			workloads, err := cluster.Workloads(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("Workloads() error = %v", err)
			}

			assertWorkloads(t, workloads, tt.want)
		})
	}
}

func assertNamespaces(t *testing.T, got []Namespace, want []Namespace) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(namespaces) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("namespaces[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertWorkloads(t *testing.T, got []Workload, want []Workload) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(workloads) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("workloads[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func newTestCluster(objects ...runtime.Object) *KubernetesCluster {
	cluster := NewKubernetesClusterWithClientset(fake.NewSimpleClientset(objects...))
	cluster.now = func() time.Time {
		return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	}
	return cluster
}
