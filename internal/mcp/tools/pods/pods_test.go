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
		input   PodsInput
		getFn   func(ctx context.Context, namespace, name string) (*corev1.Pod, error)
		wantErr bool
	}{
		{
			name:  "successful get",
			input: PodsInput{Namespace: "default", Name: "test-pod"},
			getFn: func(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
				return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
			},
			wantErr: false,
		},
		{
			name:  "not found error",
			input: PodsInput{Namespace: "default", Name: "test-pod"},
			getFn: func(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
		{
			name:    "missing namespace",
			input:   PodsInput{Name: "test-pod"},
			getFn:   func(ctx context.Context, namespace, name string) (*corev1.Pod, error) { return nil, nil },
			wantErr: true,
		},
		{
			name:    "invalid pod name",
			input:   PodsInput{Namespace: "default", Name: "Bad Pod"},
			getFn:   func(ctx context.Context, namespace, name string) (*corev1.Pod, error) { return nil, nil },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDescribePodHandler(tt.getFn)
			_, err := h.Execute(context.Background(), tt.input)
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
		input     PodLogsInput
		getLogsFn func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error)
		wantErr   bool
	}{
		{
			name:  "successful logs get",
			input: PodLogsInput{Namespace: "default", Name: "test-pod"},
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "logs", nil
			},
			wantErr: false,
		},
		{
			name:  "api error",
			input: PodLogsInput{Namespace: "default", Name: "test-pod"},
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "", errors.New("api error")
			},
			wantErr: true,
		},
		{
			name:  "negative tail lines",
			input: PodLogsInput{Namespace: "default", Name: "test-pod", TailLines: -1},
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "", nil
			},
			wantErr: true,
		},
		{
			name:  "tail lines over limit",
			input: PodLogsInput{Namespace: "default", Name: "test-pod", TailLines: 1001},
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "", nil
			},
			wantErr: true,
		},
		{
			name:  "invalid container",
			input: PodLogsInput{Namespace: "default", Name: "test-pod", Container: "bad/container"},
			getLogsFn: func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
				return "", nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGetPodLogsHandler(tt.getLogsFn)
			got, err := h.Execute(context.Background(), tt.input)
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
	negativeGraceSeconds := int64(-1)
	tooLongGraceSeconds := int64(3601)

	tests := []struct {
		name     string
		input    DeletePodInput
		deleteFn func(ctx context.Context, namespace, name string, gracePeriod *int64) error
		wantErr  bool
	}{
		{
			name:  "successful delete",
			input: DeletePodInput{Namespace: "default", Name: "test-pod"},
			deleteFn: func(ctx context.Context, namespace, name string, gracePeriod *int64) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:  "api error",
			input: DeletePodInput{Namespace: "default", Name: "test-pod"},
			deleteFn: func(ctx context.Context, namespace, name string, gracePeriod *int64) error {
				return errors.New("api error")
			},
			wantErr: true,
		},
		{
			name:     "negative grace seconds",
			input:    DeletePodInput{Namespace: "default", Name: "test-pod", GraceSeconds: &negativeGraceSeconds},
			deleteFn: func(ctx context.Context, namespace, name string, gracePeriod *int64) error { return nil },
			wantErr:  true,
		},
		{
			name:     "grace seconds over limit",
			input:    DeletePodInput{Namespace: "default", Name: "test-pod", GraceSeconds: &tooLongGraceSeconds},
			deleteFn: func(ctx context.Context, namespace, name string, gracePeriod *int64) error { return nil },
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeletePodHandler(tt.deleteFn)
			_, err := h.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
