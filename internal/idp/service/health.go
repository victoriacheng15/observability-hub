package service

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *KubernetesService) Health(ctx context.Context, opts HealthOptions) ([]Health, error) {
	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	health := make([]Health, 0, len(details))
	for _, detail := range details {
		result, err := s.health(ctx, detail)
		if err != nil {
			return nil, err
		}
		health = append(health, result)
	}

	sortHealth(health)
	return health, nil
}

func (s *KubernetesService) health(ctx context.Context, detail Details) (Health, error) {
	pods, err := s.podsFor(ctx, detail)
	if err != nil {
		return Health{}, err
	}

	readyPods := 0
	var restarts int32
	podNames := make(map[string]struct{}, len(pods))
	for _, pod := range pods {
		podNames[pod.Name] = struct{}{}
		if podReady(pod) {
			readyPods++
		}
		for _, status := range pod.Status.ContainerStatuses {
			restarts += status.RestartCount
		}
	}

	serviceCount, readyServices, err := s.serviceEndpointStatus(ctx, detail)
	if err != nil {
		return Health{}, err
	}

	warnings, err := s.warningEventReasons(ctx, detail, podNames)
	if err != nil {
		return Health{}, err
	}

	health := Health{
		Summary:             detail.Summary,
		ReadyPods:           readyPods,
		PodCount:            len(pods),
		Restarts:            restarts,
		ReadyServices:       readyServices,
		ServiceCount:        serviceCount,
		WarningEventCount:   len(warnings),
		WarningEventReasons: warnings,
	}
	health.Status = healthStatus(health)
	return health, nil
}

func (s *KubernetesService) podsFor(ctx context.Context, detail Details) ([]corev1.Pod, error) {
	if len(detail.Selector) == 0 {
		return []corev1.Pod{}, nil
	}

	pods, err := s.clientset.CoreV1().Pods(detail.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelsSelector(detail.Selector),
	})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	return pods.Items, nil
}

func (s *KubernetesService) serviceEndpointStatus(ctx context.Context, detail Details) (int, int, error) {
	if len(detail.Selector) == 0 {
		return 0, 0, nil
	}

	services, err := s.clientset.CoreV1().Services(detail.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("list services: %w", err)
	}

	serviceCount := 0
	readyServices := 0
	for _, service := range services.Items {
		if !selectorCovers(detail.Selector, service.Spec.Selector) {
			continue
		}
		serviceCount++
		ready, err := s.serviceHasReadyEndpointSlice(ctx, detail.Namespace, service.Name)
		if err != nil {
			return 0, 0, err
		}
		if ready {
			readyServices++
		}
	}

	return serviceCount, readyServices, nil
}

func (s *KubernetesService) serviceHasReadyEndpointSlice(ctx context.Context, namespace string, serviceName string) (bool, error) {
	endpointSlices, err := s.clientset.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: discoveryv1.LabelServiceName + "=" + serviceName,
	})
	if err != nil {
		return false, fmt.Errorf("list endpointslices: %w", err)
	}

	for _, endpointSlice := range endpointSlices.Items {
		for _, endpoint := range endpointSlice.Endpoints {
			if endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *KubernetesService) warningEventReasons(ctx context.Context, detail Details, podNames map[string]struct{}) ([]string, error) {
	events, err := s.clientset.CoreV1().Events(detail.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	reasons := make([]string, 0)
	for _, event := range events.Items {
		if event.Type != corev1.EventTypeWarning {
			continue
		}
		if event.InvolvedObject.Name == detail.Name {
			reasons = append(reasons, event.Reason)
			continue
		}
		if _, ok := podNames[event.InvolvedObject.Name]; ok {
			reasons = append(reasons, event.Reason)
		}
	}

	sort.Strings(reasons)
	return reasons, nil
}

func healthStatus(health Health) string {
	if health.WarningEventCount > 0 {
		return "degraded"
	}
	if !readyRatioHealthy(health.Ready) {
		return "degraded"
	}
	if health.PodCount > 0 && health.ReadyPods != health.PodCount {
		return "degraded"
	}
	if health.ServiceCount > 0 && health.ReadyServices != health.ServiceCount {
		return "degraded"
	}
	if health.PodCount == 0 && health.ServiceCount == 0 {
		return "unknown"
	}
	return "healthy"
}

func readyRatioHealthy(ready string) bool {
	var current int
	var desired int
	if _, err := fmt.Sscanf(ready, "%d/%d", &current, &desired); err != nil {
		return true
	}
	return current == desired
}
