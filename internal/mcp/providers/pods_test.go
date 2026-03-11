package providers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPodsProvider(t *testing.T) {
	t.Run("ListPods", func(t *testing.T) {
		fakePod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}
		clientset := fake.NewSimpleClientset(fakePod)
		provider := &PodsProvider{clientset: clientset}

		tests := []struct {
			name      string
			namespace string
			wantCount int
			wantErr   bool
		}{
			{
				name:      "list all pods",
				namespace: "",
				wantCount: 1,
				wantErr:   false,
			},
			{
				name:      "list pods in specific namespace",
				namespace: "default",
				wantCount: 1,
				wantErr:   false,
			},
			{
				name:      "list pods in empty namespace",
				namespace: "non-existent",
				wantCount: 0,
				wantErr:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := provider.ListPods(context.Background(), tt.namespace)
				if (err != nil) != tt.wantErr {
					t.Errorf("ListPods() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if len(got.Items) != tt.wantCount {
					t.Errorf("ListPods() got count = %v, want %v", len(got.Items), tt.wantCount)
				}
			})
		}
	})

	t.Run("GetPod", func(t *testing.T) {
		fakePod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}
		clientset := fake.NewSimpleClientset(fakePod)
		provider := &PodsProvider{clientset: clientset}

		tests := []struct {
			name      string
			namespace string
			podName   string
			wantErr   bool
		}{
			{
				name:      "get existing pod",
				namespace: "default",
				podName:   "test-pod",
				wantErr:   false,
			},
			{
				name:      "get non-existent pod",
				namespace: "default",
				podName:   "ghost-pod",
				wantErr:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := provider.GetPod(context.Background(), tt.namespace, tt.podName)
				if (err != nil) != tt.wantErr {
					t.Errorf("GetPod() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && got.Name != tt.podName {
					t.Errorf("GetPod() got = %v, want %v", got.Name, tt.podName)
				}
			})
		}
	})

	t.Run("ListEvents", func(t *testing.T) {
		fakePod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
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
			Message: "test message",
		}
		clientset := fake.NewSimpleClientset(fakePod, fakeEvent)
		provider := &PodsProvider{clientset: clientset}

		tests := []struct {
			name      string
			namespace string
			podName   string
			wantCount int
			wantErr   bool
		}{
			{
				name:      "list events for existing pod",
				namespace: "default",
				podName:   "test-pod",
				wantCount: 1,
				wantErr:   false,
			},
			{
				name:      "list events for pod with no events",
				namespace: "default",
				podName:   "quiet-pod",
				wantCount: 0,
				wantErr:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := provider.ListEvents(context.Background(), tt.namespace, tt.podName)
				if (err != nil) != tt.wantErr {
					t.Errorf("ListEvents() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && len(got.Items) != tt.wantCount {
					t.Errorf("ListEvents() got count = %v, want %v", len(got.Items), tt.wantCount)
				}
			})
		}
	})
}
