package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"observability-hub/internal/telemetry"
)

// PodsProvider provides tools to inspect and manage Kubernetes pods.
type PodsProvider struct {
	clientset kubernetes.Interface
}

// NewPodsProvider creates a new PodsProvider.
// It attempts to use in-cluster config first, then falls back to local kubeconfig.
func NewPodsProvider() (*PodsProvider, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fallback to local kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
		telemetry.Info("using local kubeconfig", "path", kubeconfig)
	} else {
		telemetry.Info("using in-cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &PodsProvider{clientset: clientset}, nil
}

// NewPodsProviderWithClientset creates a new PodsProvider with a provided clientset.
func NewPodsProviderWithClientset(clientset kubernetes.Interface) *PodsProvider {
	return &PodsProvider{clientset: clientset}
}

// ListPods returns a list of pods in the specified namespace.
func (p *PodsProvider) ListPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	pods, err := p.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	return pods, nil
}

// GetPod returns the specified pod.
func (p *PodsProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := p.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", name, namespace, err)
	}

	return pod, nil
}

// ListEvents returns a list of events for the specified pod.
func (p *PodsProvider) ListEvents(ctx context.Context, namespace, name string) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", name)
	events, err := p.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list events for pod %s/%s: %w", name, namespace, err)
	}

	// Fake clientset doesn't support field selectors, so we filter manually for consistency in tests
	filtered := &corev1.EventList{
		Items: make([]corev1.Event, 0),
	}
	for _, e := range events.Items {
		if e.InvolvedObject.Name == name && e.InvolvedObject.Kind == "Pod" {
			filtered.Items = append(filtered.Items, e)
		}
	}

	return filtered, nil
}

// GetPodLogs retrieves logs from the specified pod/container.
func (p *PodsProvider) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64, previous bool) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	req := p.clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for pod %s/%s: %w", name, namespace, err)
	}

	return string(logs), nil
}

// DeletePod deletes the specified pod.
func (p *PodsProvider) DeletePod(ctx context.Context, namespace, name string, gracePeriod *int64) error {
	opts := metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriod,
	}
	err := p.clientset.CoreV1().Pods(namespace).Delete(ctx, name, opts)
	if err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", name, namespace, err)
	}

	return nil
}
