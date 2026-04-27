package pods

import (
	"context"
	"fmt"

	toolvalidation "observability-hub/internal/mcp/tools"

	corev1 "k8s.io/api/core/v1"
)

const (
	maxLogTailLines       int64 = 1000
	maxDeleteGraceSeconds int64 = 3600
)

// PodsInput is the common input for pod-related tools.
type PodsInput struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// PodSummary represents a high-level overview of a pod for agentic analysis.
type PodSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
	Node      string `json:"node"`
}

// InspectPodsHandler handles listing pods and their health status.
type InspectPodsHandler struct {
	listFn func(ctx context.Context, namespace string) (*corev1.PodList, error)
}

func NewInspectPodsHandler(listFn func(ctx context.Context, namespace string) (*corev1.PodList, error)) *InspectPodsHandler {
	return &InspectPodsHandler{listFn: listFn}
}

func (h *InspectPodsHandler) Execute(ctx context.Context, input PodsInput) (interface{}, error) {
	if err := toolvalidation.OptionalDNS1123Label("namespace", input.Namespace); err != nil {
		return nil, err
	}

	pods, err := h.listFn(ctx, input.Namespace)
	if err != nil {
		return nil, err
	}

	summaries := make([]PodSummary, 0, len(pods.Items))
	for _, p := range pods.Items {
		summaries = append(summaries, PodSummary{
			Name:      p.Name,
			Namespace: p.Namespace,
			Status:    string(p.Status.Phase),
			IP:        p.Status.PodIP,
			Node:      p.Spec.NodeName,
		})
	}

	return summaries, nil
}

// DescribePodHandler handles getting detailed information about a pod.
type DescribePodHandler struct {
	getFn func(ctx context.Context, namespace, name string) (*corev1.Pod, error)
}

func NewDescribePodHandler(getFn func(ctx context.Context, namespace, name string) (*corev1.Pod, error)) *DescribePodHandler {
	return &DescribePodHandler{getFn: getFn}
}

func (h *DescribePodHandler) Execute(ctx context.Context, input PodsInput) (interface{}, error) {
	if err := validatePodTarget(input.Namespace, input.Name); err != nil {
		return nil, err
	}
	return h.getFn(ctx, input.Namespace, input.Name)
}

// ListPodEventsHandler handles listing events for a specific pod.
type ListPodEventsHandler struct {
	listEventsFn func(ctx context.Context, namespace, name string) (*corev1.EventList, error)
}

func NewListPodEventsHandler(listEventsFn func(ctx context.Context, namespace, name string) (*corev1.EventList, error)) *ListPodEventsHandler {
	return &ListPodEventsHandler{listEventsFn: listEventsFn}
}

func (h *ListPodEventsHandler) Execute(ctx context.Context, input PodsInput) (interface{}, error) {
	if err := validatePodTarget(input.Namespace, input.Name); err != nil {
		return nil, err
	}
	return h.listEventsFn(ctx, input.Namespace, input.Name)
}

// PodLogsInput is the input for getting pod logs.
type PodLogsInput struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Container string `json:"container,omitempty"`
	TailLines int64  `json:"tail_lines,omitempty"`
	Previous  bool   `json:"previous,omitempty"`
}

// GetPodLogsHandler handles retrieving logs for a pod.
type GetPodLogsHandler struct {
	getLogsFn func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error)
}

func NewGetPodLogsHandler(getLogsFn func(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error)) *GetPodLogsHandler {
	return &GetPodLogsHandler{getLogsFn: getLogsFn}
}

func (h *GetPodLogsHandler) Execute(ctx context.Context, input PodLogsInput) (interface{}, error) {
	if err := validatePodTarget(input.Namespace, input.Name); err != nil {
		return nil, err
	}
	if err := toolvalidation.OptionalDNS1123Label("container", input.Container); err != nil {
		return nil, err
	}
	if input.TailLines < 0 || input.TailLines > maxLogTailLines {
		return nil, fmt.Errorf("tail_lines must be between 0 and %d", maxLogTailLines)
	}
	return h.getLogsFn(ctx, input.Namespace, input.Name, input.Container, input.TailLines, input.Previous)
}

// DeletePodInput is the input for deleting a pod.
type DeletePodInput struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	GraceSeconds *int64 `json:"grace_seconds,omitempty"`
}

// DeletePodHandler handles deleting a pod.
type DeletePodHandler struct {
	deleteFn func(ctx context.Context, namespace, name string, gracePeriod *int64) error
}

func NewDeletePodHandler(deleteFn func(ctx context.Context, namespace, name string, gracePeriod *int64) error) *DeletePodHandler {
	return &DeletePodHandler{deleteFn: deleteFn}
}

func (h *DeletePodHandler) Execute(ctx context.Context, input DeletePodInput) (interface{}, error) {
	if err := validatePodTarget(input.Namespace, input.Name); err != nil {
		return nil, err
	}
	if input.GraceSeconds != nil && (*input.GraceSeconds < 0 || *input.GraceSeconds > maxDeleteGraceSeconds) {
		return nil, fmt.Errorf("grace_seconds must be between 0 and %d", maxDeleteGraceSeconds)
	}

	err := h.deleteFn(ctx, input.Namespace, input.Name, input.GraceSeconds)
	if err != nil {
		return nil, err
	}
	return map[string]string{"status": "deleted", "pod": input.Name, "namespace": input.Namespace}, nil
}

func validatePodTarget(namespace, name string) error {
	if err := toolvalidation.DNS1123Label("namespace", namespace); err != nil {
		return err
	}
	if err := toolvalidation.DNS1123Subdomain("name", name); err != nil {
		return err
	}
	return nil
}
