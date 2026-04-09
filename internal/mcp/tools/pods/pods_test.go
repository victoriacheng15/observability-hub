package pods

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInspectPodsHandler_Execute(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		listFn    func(ctx context.Context, namespace string) (*corev1.PodList, error)
		wantCount int
		wantErr   bool
	}{
		{
			name:      "successful list",
			namespace: "default",
			listFn: func(ctx context.Context, namespace string) (*corev1.PodList, error) {
				return &corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"},
							Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "1.1.1.1"},
						},
					},
				}, nil
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "empty list",
			namespace: "empty",
			listFn: func(ctx context.Context, namespace string) (*corev1.PodList, error) {
				return &corev1.PodList{Items: []corev1.Pod{}}, nil
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "provider error",
			namespace: "error",
			listFn: func(ctx context.Context, namespace string) (*corev1.PodList, error) {
				return nil, errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewInspectPodsHandler(tt.listFn)
			got, err := h.Execute(context.Background(), PodsInput{Namespace: tt.namespace})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				summaries := got.([]PodSummary)
				if len(summaries) != tt.wantCount {
					t.Errorf("Execute() got count = %v, want %v", len(summaries), tt.wantCount)
				}
			}
		})
	}
}

func TestDescribePodHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		getFn   func(ctx context.Context, namespace, name string) (*corev1.Pod, error)
		wantErr bool
	}{
		{
			name: "successful get",
			getFn: func(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
				return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
			},
			wantErr: false,
		},
		{
			name: "not found error",
			getFn: func(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDescribePodHandler(tt.getFn)
			_, err := h.Execute(context.Background(), PodsInput{Namespace: "default", Name: "test-pod"})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListPodEventsHandler_Execute(t *testing.T) {
	tests := []struct {
		name         string
		listEventsFn func(ctx context.Context, namespace, name string) (*corev1.EventList, error)
		wantErr      bool
	}{
		{
			name: "successful events list",
			listEventsFn: func(ctx context.Context, namespace, name string) (*corev1.EventList, error) {
				return &corev1.EventList{Items: []corev1.Event{{Message: "event"}}}, nil
			},
			wantErr: false,
		},
		{
			name: "api error",
			listEventsFn: func(ctx context.Context, namespace, name string) (*corev1.EventList, error) {
				return nil, errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewListPodEventsHandler(tt.listEventsFn)
			_, err := h.Execute(context.Background(), PodsInput{Namespace: "default", Name: "test-pod"})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPodLogsHandler_Execute(t *testing.T) {
	tests := []struct {
		name      string
		getLogsFn func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error)
		wantErr   bool
	}{
		{
			name: "successful logs get",
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "logs", nil
			},
			wantErr: false,
		},
		{
			name: "api error",
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "", errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGetPodLogsHandler(tt.getLogsFn)
			got, err := h.Execute(context.Background(), PodLogsInput{Namespace: "default", Name: "test-pod"})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.(string) != "logs" {
				t.Errorf("Execute() got = %v, want %v", got, "logs")
			}
		})
	}
}

func TestDeletePodHandler_Execute(t *testing.T) {
	tests := []struct {
		name     string
		deleteFn func(ctx context.Context, namespace, name string, gracePeriod *int64) error
		wantErr  bool
	}{
		{
			name: "successful delete",
			deleteFn: func(ctx context.Context, namespace, name string, gracePeriod *int64) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "api error",
			deleteFn: func(ctx context.Context, namespace, name string, gracePeriod *int64) error {
				return errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeletePodHandler(tt.deleteFn)
			_, err := h.Execute(context.Background(), DeletePodInput{Namespace: "default", Name: "test-pod"})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
