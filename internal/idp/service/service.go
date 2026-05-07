package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"observability-hub/internal/env"
)

type KubernetesService struct {
	clientset kubernetes.Interface
	now       func() time.Time
	podLogs   func(context.Context, corev1.Pod, string, LogsOptions) ([]string, error)
	signals   *signalClient
}

type Summary struct {
	Name      string
	Namespace string
	Kind      string
	Ready     string
	Age       string
}

type Details struct {
	Summary
	Images      []string
	Labels      map[string]string
	Annotations map[string]string
	Selector    map[string]string
}

type ListOptions struct {
	Namespace string
}

type DescribeOptions struct {
	Name      string
	Namespace string
}

type HealthOptions struct {
	Name      string
	Namespace string
}

type LogsOptions struct {
	Name      string
	Namespace string
	Container string
	Tail      int64
	Since     time.Duration
	Previous  bool
}

type EventsOptions struct {
	Name      string
	Namespace string
	Tail      int
	Since     time.Duration
}

type MetricsOptions struct {
	Name      string
	Namespace string
	Window    time.Duration
}

type TracesOptions struct {
	Name      string
	Namespace string
	Hours     int
	Limit     int
}

type OwnershipOptions struct {
	Name      string
	Namespace string
}

type Health struct {
	Summary
	Status              string
	ReadyPods           int
	PodCount            int
	Restarts            int32
	ReadyServices       int
	ServiceCount        int
	WarningEventCount   int
	WarningEventReasons []string
}

type LogLine struct {
	Name      string
	Namespace string
	Kind      string
	Pod       string
	Container string
	Line      string
}

type Event struct {
	Name      string
	Namespace string
	Kind      string
	Type      string
	Reason    string
	Message   string
	Object    string
	Age       string
	Time      time.Time
}

type Metric struct {
	Name      string
	Namespace string
	Kind      string
	Signal    string
	Value     string
}

type Trace struct {
	Name            string
	Namespace       string
	Kind            string
	TraceID         string
	RootServiceName string
	StartTime       string
	Duration        string
}

type Ownership struct {
	Summary
	Owner        string
	Tier         string
	Source       string
	Docs         []string
	SourceObject string
}

func NewKubernetesService() (*KubernetesService, error) {
	env.Load()

	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	return NewKubernetesServiceWithClientset(clientset), nil
}

func NewKubernetesServiceWithClientset(clientset kubernetes.Interface) *KubernetesService {
	return &KubernetesService{
		clientset: clientset,
		now:       time.Now,
		podLogs:   defaultPodLogs(clientset),
		signals:   newSignalClient(prometheusURLFromEnv(), os.Getenv("TEMPO_URL"), nil),
	}
}

func prometheusURLFromEnv() string {
	return os.Getenv("PROMETHEUS_URL")
}

func (s *KubernetesService) List(ctx context.Context, opts ListOptions) ([]Summary, error) {
	details, err := s.workloads(ctx, opts.Namespace)
	if err != nil {
		return nil, err
	}

	summaries := make([]Summary, 0, len(details))
	for _, detail := range details {
		summaries = append(summaries, detail.Summary)
	}

	sortSummaries(summaries)
	return summaries, nil
}

func (s *KubernetesService) Describe(ctx context.Context, opts DescribeOptions) ([]Details, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("missing service name")
	}

	namespace := opts.Namespace
	name := opts.Name
	if strings.Contains(opts.Name, "/") {
		parts := strings.SplitN(opts.Name, "/", 2)
		namespace = parts[0]
		name = parts[1]
	}

	workloads, err := s.workloads(ctx, namespace)
	if err != nil {
		return nil, err
	}

	matches := make([]Details, 0)
	for _, workload := range workloads {
		if serviceMatches(workload, name) {
			matches = append(matches, workload)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("service not found: %s", opts.Name)
	}

	sortDetails(matches)
	return matches, nil
}

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

func (s *KubernetesService) Events(ctx context.Context, opts EventsOptions) ([]Event, error) {
	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0)
	for _, detail := range details {
		pods, err := s.podsFor(ctx, detail)
		if err != nil {
			return nil, err
		}
		podNames := make(map[string]struct{}, len(pods))
		for _, pod := range pods {
			podNames[pod.Name] = struct{}{}
		}

		kubeEvents, err := s.clientset.CoreV1().Events(detail.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		for _, event := range kubeEvents.Items {
			if !eventMatches(detail, podNames, event) || eventOlderThan(event, opts.Since, s.now()) {
				continue
			}
			events = append(events, Event{
				Name:      detail.Name,
				Namespace: detail.Namespace,
				Kind:      detail.Kind,
				Type:      event.Type,
				Reason:    event.Reason,
				Message:   event.Message,
				Object:    event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
				Age:       s.age(eventTime(event)),
				Time:      eventTime(event),
			})
		}
	}

	sortEvents(events)
	if opts.Tail > 0 && len(events) > opts.Tail {
		events = events[:opts.Tail]
	}
	return events, nil
}

func (s *KubernetesService) Metrics(ctx context.Context, opts MetricsOptions) ([]Metric, error) {
	if s.signals == nil || s.signals.prometheusURL == "" {
		return nil, fmt.Errorf("PROMETHEUS_URL is required for service metrics")
	}
	if opts.Window <= 0 {
		opts.Window = 5 * time.Minute
	}

	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	metrics := make([]Metric, 0)
	for _, detail := range details {
		pods, err := s.podsFor(ctx, detail)
		if err != nil {
			return nil, err
		}
		if len(pods) == 0 {
			continue
		}

		podMatcher := podRegex(pods)
		queries := []struct {
			signal string
			query  string
		}{
			{
				signal: "cpu_cores",
				query:  fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace=%q,pod=~%q,container!="",container!="POD"}[%s]))`, detail.Namespace, podMatcher, promDuration(opts.Window)),
			},
			{
				signal: "memory_bytes",
				query:  fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace=%q,pod=~%q,container!="",container!="POD"})`, detail.Namespace, podMatcher),
			},
			{
				signal: "restarts",
				query:  fmt.Sprintf(`sum(kube_pod_container_status_restarts_total{namespace=%q,pod=~%q})`, detail.Namespace, podMatcher),
			},
		}

		for _, query := range queries {
			value, err := s.signals.queryMetric(ctx, query.query)
			if err != nil {
				return nil, err
			}
			metrics = append(metrics, Metric{
				Name:      detail.Name,
				Namespace: detail.Namespace,
				Kind:      detail.Kind,
				Signal:    query.signal,
				Value:     value,
			})
		}
	}

	sortMetrics(metrics)
	return metrics, nil
}

func (s *KubernetesService) Traces(ctx context.Context, opts TracesOptions) ([]Trace, error) {
	if s.signals == nil || s.signals.tempoURL == "" {
		return nil, fmt.Errorf("TEMPO_URL is required for service traces")
	}
	if opts.Hours <= 0 {
		opts.Hours = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	traces := make([]Trace, 0)
	for _, detail := range details {
		query := fmt.Sprintf(`{resource.service.name=%q}`, detail.Name)
		found, err := s.signals.searchTraces(ctx, query, opts.Hours, opts.Limit)
		if err != nil {
			return nil, err
		}
		for _, trace := range found {
			trace.Name = detail.Name
			trace.Namespace = detail.Namespace
			trace.Kind = detail.Kind
			traces = append(traces, trace)
		}
	}

	sortTraces(traces)
	if opts.Limit > 0 && len(traces) > opts.Limit {
		traces = traces[:opts.Limit]
	}
	return traces, nil
}

func (s *KubernetesService) Ownership(ctx context.Context, opts OwnershipOptions) ([]Ownership, error) {
	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	ownership := make([]Ownership, 0, len(details))
	for _, detail := range details {
		ownership = append(ownership, ownershipFor(detail))
	}

	sortOwnership(ownership)
	return ownership, nil
}

func (s *KubernetesService) workloads(ctx context.Context, namespace string) ([]Details, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	workloads := make([]Details, 0)

	deployments, err := s.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	for _, deployment := range deployments.Items {
		workloads = append(workloads, s.deploymentDetails(deployment))
	}

	statefulSets, err := s.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	for _, statefulSet := range statefulSets.Items {
		workloads = append(workloads, s.statefulSetDetails(statefulSet))
	}

	daemonSets, err := s.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list daemonsets: %w", err)
	}
	for _, daemonSet := range daemonSets.Items {
		workloads = append(workloads, s.daemonSetDetails(daemonSet))
	}

	jobs, err := s.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	for _, job := range jobs.Items {
		workloads = append(workloads, s.jobDetails(job))
	}

	cronJobs, err := s.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list cronjobs: %w", err)
	}
	for _, cronJob := range cronJobs.Items {
		workloads = append(workloads, s.cronJobDetails(cronJob))
	}

	sortDetails(workloads)
	return workloads, nil
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

func podRegex(pods []corev1.Pod) string {
	names := make([]string, 0, len(pods))
	for _, pod := range pods {
		names = append(names, regexp.QuoteMeta(pod.Name))
	}
	sort.Strings(names)
	return strings.Join(names, "|")
}

func promDuration(duration time.Duration) string {
	if duration < time.Second {
		duration = time.Second
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
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

func (s *KubernetesService) deploymentDetails(deployment appsv1.Deployment) Details {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	return Details{
		Summary: Summary{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Kind:      "Deployment",
			Ready:     fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, desired),
			Age:       s.age(deployment.CreationTimestamp.Time),
		},
		Images:      containerImages(deployment.Spec.Template.Spec),
		Labels:      copyMap(deployment.Labels),
		Annotations: copyMap(deployment.Annotations),
		Selector:    deployment.Spec.Selector.MatchLabels,
	}
}

func (s *KubernetesService) statefulSetDetails(statefulSet appsv1.StatefulSet) Details {
	desired := int32(1)
	if statefulSet.Spec.Replicas != nil {
		desired = *statefulSet.Spec.Replicas
	}

	return Details{
		Summary: Summary{
			Name:      statefulSet.Name,
			Namespace: statefulSet.Namespace,
			Kind:      "StatefulSet",
			Ready:     fmt.Sprintf("%d/%d", statefulSet.Status.ReadyReplicas, desired),
			Age:       s.age(statefulSet.CreationTimestamp.Time),
		},
		Images:      containerImages(statefulSet.Spec.Template.Spec),
		Labels:      copyMap(statefulSet.Labels),
		Annotations: copyMap(statefulSet.Annotations),
		Selector:    statefulSet.Spec.Selector.MatchLabels,
	}
}

func (s *KubernetesService) daemonSetDetails(daemonSet appsv1.DaemonSet) Details {
	return Details{
		Summary: Summary{
			Name:      daemonSet.Name,
			Namespace: daemonSet.Namespace,
			Kind:      "DaemonSet",
			Ready:     fmt.Sprintf("%d/%d", daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled),
			Age:       s.age(daemonSet.CreationTimestamp.Time),
		},
		Images:      containerImages(daemonSet.Spec.Template.Spec),
		Labels:      copyMap(daemonSet.Labels),
		Annotations: copyMap(daemonSet.Annotations),
		Selector:    daemonSet.Spec.Selector.MatchLabels,
	}
}

func (s *KubernetesService) jobDetails(job batchv1.Job) Details {
	desired := int32(1)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}

	return Details{
		Summary: Summary{
			Name:      job.Name,
			Namespace: job.Namespace,
			Kind:      "Job",
			Ready:     fmt.Sprintf("%d/%d", job.Status.Succeeded, desired),
			Age:       s.age(job.CreationTimestamp.Time),
		},
		Images:      containerImages(job.Spec.Template.Spec),
		Labels:      copyMap(job.Labels),
		Annotations: copyMap(job.Annotations),
		Selector:    map[string]string{},
	}
}

func (s *KubernetesService) cronJobDetails(cronJob batchv1.CronJob) Details {
	return Details{
		Summary: Summary{
			Name:      cronJob.Name,
			Namespace: cronJob.Namespace,
			Kind:      "CronJob",
			Ready:     fmt.Sprintf("active:%d", len(cronJob.Status.Active)),
			Age:       s.age(cronJob.CreationTimestamp.Time),
		},
		Images:      containerImages(cronJob.Spec.JobTemplate.Spec.Template.Spec),
		Labels:      copyMap(cronJob.Labels),
		Annotations: copyMap(cronJob.Annotations),
		Selector:    map[string]string{},
	}
}

func serviceMatches(detail Details, name string) bool {
	if detail.Name == name {
		return true
	}

	for _, key := range []string{"app.kubernetes.io/name", "app", "name"} {
		if detail.Labels[key] == name {
			return true
		}
		if detail.Selector[key] == name {
			return true
		}
	}

	return false
}

func containerImages(spec corev1.PodSpec) []string {
	images := make([]string, 0, len(spec.InitContainers)+len(spec.Containers))
	for _, container := range spec.InitContainers {
		images = append(images, container.Image)
	}
	for _, container := range spec.Containers {
		images = append(images, container.Image)
	}
	sort.Strings(images)
	return images
}

func copyMap(values map[string]string) map[string]string {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func sortSummaries(summaries []Summary) {
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Namespace == summaries[j].Namespace {
			if summaries[i].Kind == summaries[j].Kind {
				return summaries[i].Name < summaries[j].Name
			}
			return summaries[i].Kind < summaries[j].Kind
		}
		return summaries[i].Namespace < summaries[j].Namespace
	})
}

func sortDetails(details []Details) {
	sort.Slice(details, func(i, j int) bool {
		if details[i].Namespace == details[j].Namespace {
			if details[i].Kind == details[j].Kind {
				return details[i].Name < details[j].Name
			}
			return details[i].Kind < details[j].Kind
		}
		return details[i].Namespace < details[j].Namespace
	})
}

func sortHealth(health []Health) {
	sort.Slice(health, func(i, j int) bool {
		if health[i].Namespace == health[j].Namespace {
			if health[i].Kind == health[j].Kind {
				return health[i].Name < health[j].Name
			}
			return health[i].Kind < health[j].Kind
		}
		return health[i].Namespace < health[j].Namespace
	})
}

func sortMetrics(metrics []Metric) {
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Namespace == metrics[j].Namespace {
			if metrics[i].Name == metrics[j].Name {
				return metrics[i].Signal < metrics[j].Signal
			}
			return metrics[i].Name < metrics[j].Name
		}
		return metrics[i].Namespace < metrics[j].Namespace
	})
}

func sortTraces(traces []Trace) {
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].StartTime > traces[j].StartTime
	})
}

func sortOwnership(ownership []Ownership) {
	sort.Slice(ownership, func(i, j int) bool {
		if ownership[i].Namespace == ownership[j].Namespace {
			if ownership[i].Kind == ownership[j].Kind {
				return ownership[i].Name < ownership[j].Name
			}
			return ownership[i].Kind < ownership[j].Kind
		}
		return ownership[i].Namespace < ownership[j].Namespace
	})
}

func SortedMapEntries(values map[string]string) []string {
	entries := make([]string, 0, len(values))
	for key, value := range values {
		entries = append(entries, key+"="+value)
	}
	sort.Strings(entries)
	return entries
}

func labelsSelector(values map[string]string) string {
	parts := make([]string, 0, len(values))
	for key, value := range values {
		parts = append(parts, key+"="+value)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func selectorCovers(workloadSelector map[string]string, serviceSelector map[string]string) bool {
	if len(serviceSelector) == 0 {
		return false
	}
	for key, value := range serviceSelector {
		if workloadSelector[key] != value {
			return false
		}
	}
	return true
}

func podReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func eventMatches(detail Details, podNames map[string]struct{}, event corev1.Event) bool {
	if event.InvolvedObject.Namespace != "" && event.InvolvedObject.Namespace != detail.Namespace {
		return false
	}
	if event.InvolvedObject.Name == detail.Name {
		return true
	}
	_, ok := podNames[event.InvolvedObject.Name]
	return ok
}

func eventOlderThan(event corev1.Event, since time.Duration, now time.Time) bool {
	if since <= 0 {
		return false
	}
	return now.Sub(eventTime(event)) > since
}

func eventTime(event corev1.Event) time.Time {
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return event.CreationTimestamp.Time
}

func sortEvents(events []Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.After(events[j].Time)
	})
}

func sortPods(pods []corev1.Pod) {
	sort.Slice(pods, func(i, j int) bool {
		return pods[i].Name < pods[j].Name
	})
}

func ownershipFor(detail Details) Ownership {
	return Ownership{
		Summary:      detail.Summary,
		Owner:        metadataValue(detail, []string{"platform.observability-hub.io/owner", "app.kubernetes.io/owner", "owner", "team"}),
		Tier:         metadataValue(detail, []string{"platform.observability-hub.io/tier", "tier", "app.kubernetes.io/component", "app.kubernetes.io/part-of"}),
		Source:       metadataValue(detail, []string{"platform.observability-hub.io/source", "platform.observability-hub.io/repo", "app.kubernetes.io/source", "repository", "repo", "source"}),
		Docs:         metadataValues(detail, []string{"platform.observability-hub.io/docs", "platform.observability-hub.io/runbook", "platform.observability-hub.io/dashboard", "docs", "documentation", "runbook", "dashboard"}),
		SourceObject: detail.Kind + " " + detail.Namespace + "/" + detail.Name,
	}
}

func metadataValue(detail Details, keys []string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(detail.Labels[key]); value != "" {
			return value
		}
		if value := strings.TrimSpace(detail.Annotations[key]); value != "" {
			return value
		}
	}
	return "unknown"
}

func metadataValues(detail Details, keys []string) []string {
	values := make([]string, 0)
	for _, key := range keys {
		if value := strings.TrimSpace(detail.Labels[key]); value != "" {
			values = append(values, key+"="+value)
		}
		if value := strings.TrimSpace(detail.Annotations[key]); value != "" {
			values = append(values, key+"="+value)
		}
	}
	if len(values) == 0 {
		return []string{"unknown"}
	}
	sort.Strings(values)
	return values
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

type signalClient struct {
	prometheusURL string
	tempoURL      string
	httpClient    *http.Client
	now           func() time.Time
}

type promQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type tempoSearchResponse struct {
	Traces []tempoTrace `json:"traces"`
}

type tempoTrace struct {
	TraceID           string `json:"traceID"`
	RootServiceName   string `json:"rootServiceName"`
	StartTimeUnixNano string `json:"startTimeUnixNano"`
	DurationMs        int64  `json:"durationMs"`
}

func newSignalClient(prometheusURL string, tempoURL string, httpClient *http.Client) *signalClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &signalClient{
		prometheusURL: strings.TrimRight(prometheusURL, "/"),
		tempoURL:      strings.TrimRight(tempoURL, "/"),
		httpClient:    httpClient,
		now:           time.Now,
	}
}

func (c *signalClient) queryMetric(ctx context.Context, query string) (string, error) {
	params := url.Values{}
	params.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.prometheusURL+"/api/v1/query?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("create metrics request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("query metrics: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return "", fmt.Errorf("query metrics: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded promQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode metrics response: %w", err)
	}
	if decoded.Status != "success" {
		return "", fmt.Errorf("query metrics: prometheus status %q", decoded.Status)
	}
	if len(decoded.Data.Result) == 0 || len(decoded.Data.Result[0].Value) < 2 {
		return "n/a", nil
	}

	value, ok := decoded.Data.Result[0].Value[1].(string)
	if !ok || value == "" {
		return "n/a", nil
	}
	return value, nil
}

func (c *signalClient) searchTraces(ctx context.Context, query string, hours int, limit int) ([]Trace, error) {
	if hours <= 0 {
		hours = 1
	}
	if hours > 168 {
		hours = 168
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	now := c.now()
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("start", strconv.FormatInt(now.Add(-time.Duration(hours)*time.Hour).Unix(), 10))
	params.Set("end", strconv.FormatInt(now.Unix(), 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.tempoURL+"/api/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create traces request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query traces: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("query traces: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded tempoSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode traces response: %w", err)
	}

	traces := make([]Trace, 0, len(decoded.Traces))
	for _, item := range decoded.Traces {
		traces = append(traces, Trace{
			TraceID:         item.TraceID,
			RootServiceName: item.RootServiceName,
			StartTime:       formatUnixNano(item.StartTimeUnixNano),
			Duration:        formatDurationMs(item.DurationMs),
		})
	}
	return traces, nil
}

func formatUnixNano(value string) string {
	if value == "" {
		return "unknown"
	}
	nanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value
	}
	return time.Unix(0, nanos).UTC().Format(time.RFC3339)
}

func formatDurationMs(value int64) string {
	if value <= 0 {
		return "unknown"
	}
	return (time.Duration(value) * time.Millisecond).String()
}

func readyRatioHealthy(ready string) bool {
	var current int
	var desired int
	if _, err := fmt.Sscanf(ready, "%d/%d", &current, &desired); err != nil {
		return true
	}
	return current == desired
}

func (s *KubernetesService) age(created time.Time) string {
	if created.IsZero() {
		return "unknown"
	}

	duration := s.now().Sub(created)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}
