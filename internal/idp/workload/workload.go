package workload

import (
	"fmt"
	"sort"
	"time"

	"observability-hub/internal/idp/kube"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

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

func DeploymentSummary(deployment appsv1.Deployment, now time.Time) Summary {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	return Summary{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Kind:      "Deployment",
		Ready:     fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, desired),
		Age:       kube.Age(now, deployment.CreationTimestamp.Time),
	}
}

func StatefulSetSummary(statefulSet appsv1.StatefulSet, now time.Time) Summary {
	desired := int32(1)
	if statefulSet.Spec.Replicas != nil {
		desired = *statefulSet.Spec.Replicas
	}

	return Summary{
		Name:      statefulSet.Name,
		Namespace: statefulSet.Namespace,
		Kind:      "StatefulSet",
		Ready:     fmt.Sprintf("%d/%d", statefulSet.Status.ReadyReplicas, desired),
		Age:       kube.Age(now, statefulSet.CreationTimestamp.Time),
	}
}

func DaemonSetSummary(daemonSet appsv1.DaemonSet, now time.Time) Summary {
	return Summary{
		Name:      daemonSet.Name,
		Namespace: daemonSet.Namespace,
		Kind:      "DaemonSet",
		Ready:     fmt.Sprintf("%d/%d", daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled),
		Age:       kube.Age(now, daemonSet.CreationTimestamp.Time),
	}
}

func JobSummary(job batchv1.Job, now time.Time) Summary {
	desired := int32(1)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}

	return Summary{
		Name:      job.Name,
		Namespace: job.Namespace,
		Kind:      "Job",
		Ready:     fmt.Sprintf("%d/%d", job.Status.Succeeded, desired),
		Age:       kube.Age(now, job.CreationTimestamp.Time),
	}
}

func CronJobSummary(cronJob batchv1.CronJob, now time.Time) Summary {
	return Summary{
		Name:      cronJob.Name,
		Namespace: cronJob.Namespace,
		Kind:      "CronJob",
		Ready:     fmt.Sprintf("active:%d", len(cronJob.Status.Active)),
		Age:       kube.Age(now, cronJob.CreationTimestamp.Time),
	}
}

func DeploymentDetails(deployment appsv1.Deployment, now time.Time) Details {
	return Details{
		Summary:     DeploymentSummary(deployment, now),
		Images:      containerImages(deployment.Spec.Template.Spec),
		Labels:      copyMap(deployment.Labels),
		Annotations: copyMap(deployment.Annotations),
		Selector:    deployment.Spec.Selector.MatchLabels,
	}
}

func StatefulSetDetails(statefulSet appsv1.StatefulSet, now time.Time) Details {
	return Details{
		Summary:     StatefulSetSummary(statefulSet, now),
		Images:      containerImages(statefulSet.Spec.Template.Spec),
		Labels:      copyMap(statefulSet.Labels),
		Annotations: copyMap(statefulSet.Annotations),
		Selector:    statefulSet.Spec.Selector.MatchLabels,
	}
}

func DaemonSetDetails(daemonSet appsv1.DaemonSet, now time.Time) Details {
	return Details{
		Summary:     DaemonSetSummary(daemonSet, now),
		Images:      containerImages(daemonSet.Spec.Template.Spec),
		Labels:      copyMap(daemonSet.Labels),
		Annotations: copyMap(daemonSet.Annotations),
		Selector:    daemonSet.Spec.Selector.MatchLabels,
	}
}

func JobDetails(job batchv1.Job, now time.Time) Details {
	return Details{
		Summary:     JobSummary(job, now),
		Images:      containerImages(job.Spec.Template.Spec),
		Labels:      copyMap(job.Labels),
		Annotations: copyMap(job.Annotations),
		Selector:    map[string]string{},
	}
}

func CronJobDetails(cronJob batchv1.CronJob, now time.Time) Details {
	return Details{
		Summary:     CronJobSummary(cronJob, now),
		Images:      containerImages(cronJob.Spec.JobTemplate.Spec.Template.Spec),
		Labels:      copyMap(cronJob.Labels),
		Annotations: copyMap(cronJob.Annotations),
		Selector:    map[string]string{},
	}
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
