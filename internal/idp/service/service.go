package service

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"observability-hub/internal/env"
	"observability-hub/internal/idp/kube"
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

	clientset, err := kube.NewClientset()
	if err != nil {
		return nil, err
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
