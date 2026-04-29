package env

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesEnvironmentList(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    ListOptions
		want    []Resource
	}{
		{
			name: "lists configmaps and secrets",
			objects: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana-env", Namespace: "observability", CreationTimestamp: created},
					Data:       map[string]string{"GF_SERVER_ROOT_URL": "http://grafana"},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana-admin", Namespace: "observability", CreationTimestamp: created},
					Type:       corev1.SecretTypeOpaque,
					Data:       map[string][]byte{"password": []byte("redacted")},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "payments-env", Namespace: "payments", CreationTimestamp: created},
					Data:       map[string]string{"DATABASE_HOST": "postgres"},
				},
			},
			want: []Resource{
				{Name: "grafana-env", Namespace: "observability", Kind: "ConfigMap", Type: "-", DataCount: 1, Age: "2h"},
				{Name: "grafana-admin", Namespace: "observability", Kind: "Secret", Type: "Opaque", DataCount: 1, Age: "2h"},
				{Name: "payments-env", Namespace: "payments", Kind: "ConfigMap", Type: "-", DataCount: 1, Age: "2h"},
			},
		},
		{
			name: "filters by namespace",
			objects: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana-env", Namespace: "observability", CreationTimestamp: created},
					Data:       map[string]string{"GF_SERVER_ROOT_URL": "http://grafana"},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "payments-token", Namespace: "payments", CreationTimestamp: created},
					Type:       corev1.SecretTypeOpaque,
					Data:       map[string][]byte{"token": []byte("redacted")},
				},
			},
			opts: ListOptions{Namespace: "payments"},
			want: []Resource{
				{Name: "payments-token", Namespace: "payments", Kind: "Secret", Type: "Opaque", DataCount: 1, Age: "2h"},
			},
		},
		{
			name: "empty env resources",
			want: []Resource{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := newTestEnvironment(tt.objects...)

			resources, err := environment.List(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			assertResources(t, resources, tt.want)
		})
	}
}

func TestKubernetesEnvironmentDescribe(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))

	tests := []struct {
		name    string
		objects []runtime.Object
		opts    DescribeOptions
		want    []Details
		wantErr string
	}{
		{
			name: "describes configmap keys",
			objects: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana-env", Namespace: "observability", CreationTimestamp: created},
					Data: map[string]string{
						"GF_SECURITY_ADMIN_USER": "admin",
						"GF_SERVER_ROOT_URL":     "http://grafana",
					},
				},
			},
			opts: DescribeOptions{Name: "grafana-env", Namespace: "observability"},
			want: []Details{
				{
					Resource: Resource{Name: "grafana-env", Namespace: "observability", Kind: "ConfigMap", Type: "-", DataCount: 2, Age: "2h"},
					Keys:     []string{"GF_SECURITY_ADMIN_USER", "GF_SERVER_ROOT_URL"},
				},
			},
		},
		{
			name: "describes secret keys without values",
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "grafana-admin", Namespace: "observability", CreationTimestamp: created},
					Type:       corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"password": []byte("redacted"),
						"username": []byte("admin"),
					},
				},
			},
			opts: DescribeOptions{Name: "grafana-admin", Kind: "Secret"},
			want: []Details{
				{
					Resource: Resource{Name: "grafana-admin", Namespace: "observability", Kind: "Secret", Type: "Opaque", DataCount: 2, Age: "2h"},
					Keys:     []string{"password", "username"},
				},
			},
		},
		{
			name:    "missing resource",
			opts:    DescribeOptions{Name: "missing"},
			wantErr: "environment resource not found: missing",
		},
		{
			name: "describes duplicate names across namespaces",
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "observability", CreationTimestamp: created},
					Type:       corev1.SecretTypeOpaque,
					Data:       map[string][]byte{"password": []byte("redacted")},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "payments", CreationTimestamp: created},
					Type:       corev1.SecretTypeOpaque,
					Data:       map[string][]byte{"token": []byte("redacted")},
				},
			},
			opts: DescribeOptions{Name: "shared"},
			want: []Details{
				{
					Resource: Resource{Name: "shared", Namespace: "observability", Kind: "Secret", Type: "Opaque", DataCount: 1, Age: "2h"},
					Keys:     []string{"password"},
				},
				{
					Resource: Resource{Name: "shared", Namespace: "payments", Kind: "Secret", Type: "Opaque", DataCount: 1, Age: "2h"},
					Keys:     []string{"token"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := newTestEnvironment(tt.objects...)

			details, err := environment.Describe(context.Background(), tt.opts)
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

func assertResources(t *testing.T, got []Resource, want []Resource) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(resources) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resources[%d] = %#v, want %#v", i, got[i], want[i])
		}
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

func assertDetails(t *testing.T, got []Details, want []Details) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(details) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Resource != want[i].Resource {
			t.Fatalf("details[%d].Resource = %#v, want %#v", i, got[i].Resource, want[i].Resource)
		}
		assertStringSlice(t, got[i].Keys, want[i].Keys)
	}
}

func newTestEnvironment(objects ...runtime.Object) *KubernetesEnvironment {
	environment := NewKubernetesEnvironmentWithClientset(fake.NewSimpleClientset(objects...))
	environment.now = func() time.Time {
		return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	}
	return environment
}
