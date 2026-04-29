package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesCatalog struct {
	clientset kubernetes.Interface
	now       func() time.Time
}

type Entry struct {
	Name      string
	Namespace string
	Kind      string
	Status    string
	Age       string
}

type Validation struct {
	ResourceCount  int
	NamespaceCount int
	KindCounts     map[string]int
}

type ListOptions struct {
	Namespace string
}

func NewKubernetesCatalog() (*KubernetesCatalog, error) {
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

	return NewKubernetesCatalogWithClientset(clientset), nil
}

func NewKubernetesCatalogWithClientset(clientset kubernetes.Interface) *KubernetesCatalog {
	return &KubernetesCatalog{
		clientset: clientset,
		now:       time.Now,
	}
}

func (c *KubernetesCatalog) List(ctx context.Context, opts ListOptions) ([]Entry, error) {
	entries := make([]Entry, 0)
	namespace := opts.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	services, err := c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	for _, service := range services.Items {
		entries = append(entries, c.serviceEntry(service))
	}

	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	for _, deployment := range deployments.Items {
		entries = append(entries, c.deploymentEntry(deployment))
	}

	statefulSets, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	for _, statefulSet := range statefulSets.Items {
		entries = append(entries, c.statefulSetEntry(statefulSet))
	}

	daemonSets, err := c.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list daemonsets: %w", err)
	}
	for _, daemonSet := range daemonSets.Items {
		entries = append(entries, c.daemonSetEntry(daemonSet))
	}

	jobs, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	for _, job := range jobs.Items {
		entries = append(entries, c.jobEntry(job))
	}

	cronJobs, err := c.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list cronjobs: %w", err)
	}
	for _, cronJob := range cronJobs.Items {
		entries = append(entries, c.cronJobEntry(cronJob))
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Namespace == entries[j].Namespace {
			if entries[i].Kind == entries[j].Kind {
				return entries[i].Name < entries[j].Name
			}
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Namespace < entries[j].Namespace
	})

	return entries, nil
}

func (c *KubernetesCatalog) Validate(ctx context.Context, opts ListOptions) (Validation, error) {
	entries, err := c.List(ctx, opts)
	if err != nil {
		return Validation{}, err
	}

	namespaces := make(map[string]struct{})
	kinds := make(map[string]int)
	for _, entry := range entries {
		namespaces[entry.Namespace] = struct{}{}
		kinds[entry.Kind]++
	}

	return Validation{
		ResourceCount:  len(entries),
		NamespaceCount: len(namespaces),
		KindCounts:     kinds,
	}, nil
}

func (c *KubernetesCatalog) serviceEntry(service corev1.Service) Entry {
	return Entry{
		Name:      service.Name,
		Namespace: service.Namespace,
		Kind:      "Service",
		Status:    string(service.Spec.Type),
		Age:       c.age(service.CreationTimestamp.Time),
	}
}

func (c *KubernetesCatalog) deploymentEntry(deployment appsv1.Deployment) Entry {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	return Entry{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Kind:      "Deployment",
		Status:    fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, desired),
		Age:       c.age(deployment.CreationTimestamp.Time),
	}
}

func (c *KubernetesCatalog) statefulSetEntry(statefulSet appsv1.StatefulSet) Entry {
	desired := int32(1)
	if statefulSet.Spec.Replicas != nil {
		desired = *statefulSet.Spec.Replicas
	}

	return Entry{
		Name:      statefulSet.Name,
		Namespace: statefulSet.Namespace,
		Kind:      "StatefulSet",
		Status:    fmt.Sprintf("%d/%d", statefulSet.Status.ReadyReplicas, desired),
		Age:       c.age(statefulSet.CreationTimestamp.Time),
	}
}

func (c *KubernetesCatalog) daemonSetEntry(daemonSet appsv1.DaemonSet) Entry {
	return Entry{
		Name:      daemonSet.Name,
		Namespace: daemonSet.Namespace,
		Kind:      "DaemonSet",
		Status:    fmt.Sprintf("%d/%d", daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled),
		Age:       c.age(daemonSet.CreationTimestamp.Time),
	}
}

func (c *KubernetesCatalog) jobEntry(job batchv1.Job) Entry {
	desired := int32(1)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}

	return Entry{
		Name:      job.Name,
		Namespace: job.Namespace,
		Kind:      "Job",
		Status:    fmt.Sprintf("%d/%d", job.Status.Succeeded, desired),
		Age:       c.age(job.CreationTimestamp.Time),
	}
}

func (c *KubernetesCatalog) cronJobEntry(cronJob batchv1.CronJob) Entry {
	return Entry{
		Name:      cronJob.Name,
		Namespace: cronJob.Namespace,
		Kind:      "CronJob",
		Status:    fmt.Sprintf("active:%d", len(cronJob.Status.Active)),
		Age:       c.age(cronJob.CreationTimestamp.Time),
	}
}

func (c *KubernetesCatalog) age(created time.Time) string {
	if created.IsZero() {
		return "unknown"
	}

	duration := c.now().Sub(created)
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
