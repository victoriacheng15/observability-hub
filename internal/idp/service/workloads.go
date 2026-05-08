package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
