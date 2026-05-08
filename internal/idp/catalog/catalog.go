package catalog

import (
	"context"
	"fmt"
	"sort"
	"time"

	"observability-hub/internal/idp/kube"
	"observability-hub/internal/idp/workload"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	clientset, err := kube.NewClientset()
	if err != nil {
		return nil, err
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
	summary := workload.DeploymentSummary(deployment, c.now())

	return Entry{
		Name:      summary.Name,
		Namespace: summary.Namespace,
		Kind:      summary.Kind,
		Status:    summary.Ready,
		Age:       summary.Age,
	}
}

func (c *KubernetesCatalog) statefulSetEntry(statefulSet appsv1.StatefulSet) Entry {
	summary := workload.StatefulSetSummary(statefulSet, c.now())

	return Entry{
		Name:      summary.Name,
		Namespace: summary.Namespace,
		Kind:      summary.Kind,
		Status:    summary.Ready,
		Age:       summary.Age,
	}
}

func (c *KubernetesCatalog) daemonSetEntry(daemonSet appsv1.DaemonSet) Entry {
	summary := workload.DaemonSetSummary(daemonSet, c.now())

	return Entry{
		Name:      summary.Name,
		Namespace: summary.Namespace,
		Kind:      summary.Kind,
		Status:    summary.Ready,
		Age:       summary.Age,
	}
}

func (c *KubernetesCatalog) jobEntry(job batchv1.Job) Entry {
	summary := workload.JobSummary(job, c.now())

	return Entry{
		Name:      summary.Name,
		Namespace: summary.Namespace,
		Kind:      summary.Kind,
		Status:    summary.Ready,
		Age:       summary.Age,
	}
}

func (c *KubernetesCatalog) cronJobEntry(cronJob batchv1.CronJob) Entry {
	summary := workload.CronJobSummary(cronJob, c.now())

	return Entry{
		Name:      summary.Name,
		Namespace: summary.Namespace,
		Kind:      summary.Kind,
		Status:    summary.Ready,
		Age:       summary.Age,
	}
}

func (c *KubernetesCatalog) age(created time.Time) string {
	return kube.Age(c.now(), created)
}
