package service

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (s *KubernetesService) Logs(ctx context.Context, opts LogsOptions) ([]LogLine, error) {
	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		if err.Error() != "service not found: "+opts.Name {
			return nil, err
		}
		return s.logsForPodName(ctx, opts)
	}

	lines := make([]LogLine, 0)
	for _, detail := range details {
		pods, err := s.podsFor(ctx, detail)
		if err != nil {
			return nil, err
		}
		sortPods(pods)

		for _, pod := range pods {
			lines, err = appendPodLogs(ctx, s, lines, detail.Name, detail.Namespace, detail.Kind, pod, opts)
			if err != nil {
				return nil, err
			}
		}
	}

	return lines, nil
}

func (s *KubernetesService) logsForPodName(ctx context.Context, opts LogsOptions) ([]LogLine, error) {
	namespace := opts.Namespace
	name := opts.Name
	if strings.Contains(opts.Name, "/") {
		parts := strings.SplitN(opts.Name, "/", 2)
		namespace = parts[0]
		name = parts[1]
	}
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	matches := make([]corev1.Pod, 0)
	for _, pod := range pods.Items {
		if pod.Name == name {
			matches = append(matches, pod)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("service not found: %s", opts.Name)
	}
	sortPods(matches)

	lines := make([]LogLine, 0)
	for _, pod := range matches {
		var err error
		lines, err = appendPodLogs(ctx, s, lines, pod.Name, pod.Namespace, "Pod", pod, opts)
		if err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func appendPodLogs(ctx context.Context, service *KubernetesService, lines []LogLine, name string, namespace string, kind string, pod corev1.Pod, opts LogsOptions) ([]LogLine, error) {
	for _, container := range logContainers(pod, opts.Container) {
		podLines, err := service.podLogs(ctx, pod, container, opts)
		if err != nil {
			return nil, err
		}
		for _, line := range podLines {
			lines = append(lines, LogLine{
				Name:      name,
				Namespace: namespace,
				Kind:      kind,
				Pod:       pod.Name,
				Container: container,
				Line:      line,
			})
		}
	}
	return lines, nil
}

func defaultPodLogs(clientset kubernetes.Interface) func(context.Context, corev1.Pod, string, LogsOptions) ([]string, error) {
	return func(ctx context.Context, pod corev1.Pod, container string, opts LogsOptions) ([]string, error) {
		return readPodLogs(ctx, clientset, pod, container, opts)
	}
}

func readPodLogs(ctx context.Context, clientset kubernetes.Interface, pod corev1.Pod, container string, opts LogsOptions) ([]string, error) {
	podLogOptions := &corev1.PodLogOptions{
		Container: container,
		Previous:  opts.Previous,
	}
	if opts.Tail > 0 {
		podLogOptions.TailLines = &opts.Tail
	}
	if opts.Since > 0 {
		sinceSeconds := int64(opts.Since.Seconds())
		podLogOptions.SinceSeconds = &sinceSeconds
	}

	stream, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("read logs for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	defer stream.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan logs for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	return lines, nil
}

func logContainers(pod corev1.Pod, requested string) []string {
	if requested != "" {
		return []string{requested}
	}

	containers := make([]string, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}
	if len(containers) == 0 {
		return []string{""}
	}
	sort.Strings(containers)
	return containers
}
