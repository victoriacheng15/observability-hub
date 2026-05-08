package cluster

import (
	"context"
	"fmt"
	"sort"
	"time"

	"observability-hub/internal/idp/kube"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubernetesCluster struct {
	clientset kubernetes.Interface
	now       func() time.Time
}

type Status struct {
	ReadyNodes     int
	NodeCount      int
	NamespaceCount int
	WorkloadCount  int
}

type Namespace struct {
	Name  string
	Phase string
	Age   string
}

type Workload struct {
	Name      string
	Namespace string
	Kind      string
	Ready     string
	Age       string
}

type WorkloadOptions struct {
	Namespace string
}

func NewKubernetesCluster() (*KubernetesCluster, error) {
	clientset, err := kube.NewClientset()
	if err != nil {
		return nil, err
	}

	return NewKubernetesClusterWithClientset(clientset), nil
}

func NewKubernetesClusterWithClientset(clientset kubernetes.Interface) *KubernetesCluster {
	return &KubernetesCluster{
		clientset: clientset,
		now:       time.Now,
	}
}

func (c *KubernetesCluster) Status(ctx context.Context) (Status, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return Status{}, fmt.Errorf("list nodes: %w", err)
	}

	namespaces, err := c.Namespaces(ctx)
	if err != nil {
		return Status{}, err
	}

	workloads, err := c.Workloads(ctx, WorkloadOptions{})
	if err != nil {
		return Status{}, err
	}

	readyNodes := 0
	for _, node := range nodes.Items {
		if nodeReady(node) {
			readyNodes++
		}
	}

	return Status{
		ReadyNodes:     readyNodes,
		NodeCount:      len(nodes.Items),
		NamespaceCount: len(namespaces),
		WorkloadCount:  len(workloads),
	}, nil
}

func (c *KubernetesCluster) Namespaces(ctx context.Context) ([]Namespace, error) {
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	entries := make([]Namespace, 0, len(namespaces.Items))
	for _, namespace := range namespaces.Items {
		entries = append(entries, Namespace{
			Name:  namespace.Name,
			Phase: string(namespace.Status.Phase),
			Age:   c.age(namespace.CreationTimestamp.Time),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

func (c *KubernetesCluster) Workloads(ctx context.Context, opts WorkloadOptions) ([]Workload, error) {
	workloads := make([]Workload, 0)
	namespace := opts.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	for _, deployment := range deployments.Items {
		workloads = append(workloads, c.deploymentWorkload(deployment))
	}

	statefulSets, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	for _, statefulSet := range statefulSets.Items {
		workloads = append(workloads, c.statefulSetWorkload(statefulSet))
	}

	daemonSets, err := c.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list daemonsets: %w", err)
	}
	for _, daemonSet := range daemonSets.Items {
		workloads = append(workloads, c.daemonSetWorkload(daemonSet))
	}

	jobs, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	for _, job := range jobs.Items {
		workloads = append(workloads, c.jobWorkload(job))
	}

	cronJobs, err := c.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list cronjobs: %w", err)
	}
	for _, cronJob := range cronJobs.Items {
		workloads = append(workloads, c.cronJobWorkload(cronJob))
	}

	sort.Slice(workloads, func(i, j int) bool {
		if workloads[i].Namespace == workloads[j].Namespace {
			if workloads[i].Kind == workloads[j].Kind {
				return workloads[i].Name < workloads[j].Name
			}
			return workloads[i].Kind < workloads[j].Kind
		}
		return workloads[i].Namespace < workloads[j].Namespace
	})

	return workloads, nil
}

func nodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (c *KubernetesCluster) deploymentWorkload(deployment appsv1.Deployment) Workload {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	return Workload{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Kind:      "Deployment",
		Ready:     fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, desired),
		Age:       c.age(deployment.CreationTimestamp.Time),
	}
}

func (c *KubernetesCluster) statefulSetWorkload(statefulSet appsv1.StatefulSet) Workload {
	desired := int32(1)
	if statefulSet.Spec.Replicas != nil {
		desired = *statefulSet.Spec.Replicas
	}

	return Workload{
		Name:      statefulSet.Name,
		Namespace: statefulSet.Namespace,
		Kind:      "StatefulSet",
		Ready:     fmt.Sprintf("%d/%d", statefulSet.Status.ReadyReplicas, desired),
		Age:       c.age(statefulSet.CreationTimestamp.Time),
	}
}

func (c *KubernetesCluster) daemonSetWorkload(daemonSet appsv1.DaemonSet) Workload {
	return Workload{
		Name:      daemonSet.Name,
		Namespace: daemonSet.Namespace,
		Kind:      "DaemonSet",
		Ready:     fmt.Sprintf("%d/%d", daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled),
		Age:       c.age(daemonSet.CreationTimestamp.Time),
	}
}

func (c *KubernetesCluster) jobWorkload(job batchv1.Job) Workload {
	desired := int32(1)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}

	return Workload{
		Name:      job.Name,
		Namespace: job.Namespace,
		Kind:      "Job",
		Ready:     fmt.Sprintf("%d/%d", job.Status.Succeeded, desired),
		Age:       c.age(job.CreationTimestamp.Time),
	}
}

func (c *KubernetesCluster) cronJobWorkload(cronJob batchv1.CronJob) Workload {
	return Workload{
		Name:      cronJob.Name,
		Namespace: cronJob.Namespace,
		Kind:      "CronJob",
		Ready:     fmt.Sprintf("active:%d", len(cronJob.Status.Active)),
		Age:       c.age(cronJob.CreationTimestamp.Time),
	}
}

func (c *KubernetesCluster) age(created time.Time) string {
	return kube.Age(c.now(), created)
}
