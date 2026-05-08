package service

import (
	"context"
	"fmt"
	"strings"

	"observability-hub/internal/idp/workload"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	return serviceDetails(workload.DeploymentDetails(deployment, s.now()))
}

func (s *KubernetesService) statefulSetDetails(statefulSet appsv1.StatefulSet) Details {
	return serviceDetails(workload.StatefulSetDetails(statefulSet, s.now()))
}

func (s *KubernetesService) daemonSetDetails(daemonSet appsv1.DaemonSet) Details {
	return serviceDetails(workload.DaemonSetDetails(daemonSet, s.now()))
}

func (s *KubernetesService) jobDetails(job batchv1.Job) Details {
	return serviceDetails(workload.JobDetails(job, s.now()))
}

func (s *KubernetesService) cronJobDetails(cronJob batchv1.CronJob) Details {
	return serviceDetails(workload.CronJobDetails(cronJob, s.now()))
}

func serviceDetails(detail workload.Details) Details {
	return Details{
		Summary: Summary{
			Name:      detail.Name,
			Namespace: detail.Namespace,
			Kind:      detail.Kind,
			Ready:     detail.Ready,
			Age:       detail.Age,
		},
		Images:      detail.Images,
		Labels:      detail.Labels,
		Annotations: detail.Annotations,
		Selector:    detail.Selector,
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
