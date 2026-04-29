package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesEnvironment struct {
	clientset kubernetes.Interface
	now       func() time.Time
}

type Resource struct {
	Name      string
	Namespace string
	Kind      string
	Type      string
	DataCount int
	Age       string
}

type Details struct {
	Resource
	Keys []string
}

type ListOptions struct {
	Namespace string
}

type DescribeOptions struct {
	Name      string
	Namespace string
	Kind      string
}

func NewKubernetesEnvironment() (*KubernetesEnvironment, error) {
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

	return NewKubernetesEnvironmentWithClientset(clientset), nil
}

func NewKubernetesEnvironmentWithClientset(clientset kubernetes.Interface) *KubernetesEnvironment {
	return &KubernetesEnvironment{
		clientset: clientset,
		now:       time.Now,
	}
}

func (e *KubernetesEnvironment) List(ctx context.Context, opts ListOptions) ([]Resource, error) {
	namespace := opts.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	resources := make([]Resource, 0)

	configMaps, err := e.clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list configmaps: %w", err)
	}
	for _, configMap := range configMaps.Items {
		resources = append(resources, e.configMapResource(configMap))
	}

	secrets, err := e.clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	for _, secret := range secrets.Items {
		resources = append(resources, e.secretResource(secret))
	}

	sortResources(resources)
	return resources, nil
}

func (e *KubernetesEnvironment) Describe(ctx context.Context, opts DescribeOptions) ([]Details, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("missing environment resource name")
	}

	resources, err := e.List(ctx, ListOptions{Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	matches := make([]Resource, 0)
	for _, resource := range resources {
		if resource.Name != opts.Name {
			continue
		}
		if opts.Kind != "" && !strings.EqualFold(resource.Kind, opts.Kind) {
			continue
		}
		matches = append(matches, resource)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("environment resource not found: %s", opts.Name)
	}

	details := make([]Details, 0, len(matches))
	for _, match := range matches {
		detail, err := e.details(ctx, match)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}

	return details, nil
}

func (e *KubernetesEnvironment) details(ctx context.Context, resource Resource) (Details, error) {
	switch resource.Kind {
	case "ConfigMap":
		configMap, err := e.clientset.CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
		if err != nil {
			return Details{}, fmt.Errorf("get configmap: %w", err)
		}
		return Details{Resource: resource, Keys: sortedStringKeys(configMap.Data)}, nil
	case "Secret":
		secret, err := e.clientset.CoreV1().Secrets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
		if err != nil {
			return Details{}, fmt.Errorf("get secret: %w", err)
		}
		return Details{Resource: resource, Keys: sortedByteKeys(secret.Data)}, nil
	default:
		return Details{}, fmt.Errorf("unsupported environment resource kind: %s", resource.Kind)
	}
}

func (e *KubernetesEnvironment) configMapResource(configMap corev1.ConfigMap) Resource {
	return Resource{
		Name:      configMap.Name,
		Namespace: configMap.Namespace,
		Kind:      "ConfigMap",
		Type:      "-",
		DataCount: len(configMap.Data),
		Age:       e.age(configMap.CreationTimestamp.Time),
	}
}

func (e *KubernetesEnvironment) secretResource(secret corev1.Secret) Resource {
	return Resource{
		Name:      secret.Name,
		Namespace: secret.Namespace,
		Kind:      "Secret",
		Type:      string(secret.Type),
		DataCount: len(secret.Data),
		Age:       e.age(secret.CreationTimestamp.Time),
	}
}

func sortResources(resources []Resource) {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Namespace == resources[j].Namespace {
			if resources[i].Kind == resources[j].Kind {
				return resources[i].Name < resources[j].Name
			}
			return resources[i].Kind < resources[j].Kind
		}
		return resources[i].Namespace < resources[j].Namespace
	})
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedByteKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (e *KubernetesEnvironment) age(created time.Time) string {
	if created.IsZero() {
		return "unknown"
	}

	duration := e.now().Sub(created)
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
