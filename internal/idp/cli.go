package idp

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"observability-hub/internal/idp/catalog"
	"observability-hub/internal/idp/cluster"
	idpenv "observability-hub/internal/idp/env"
	idpservice "observability-hub/internal/idp/service"
)

type catalogClient interface {
	List(context.Context, catalog.ListOptions) ([]catalog.Entry, error)
	Validate(context.Context, catalog.ListOptions) (catalog.Validation, error)
}

type clusterClient interface {
	Status(context.Context) (cluster.Status, error)
	Namespaces(context.Context) ([]cluster.Namespace, error)
	Workloads(context.Context, cluster.WorkloadOptions) ([]cluster.Workload, error)
}

type envClient interface {
	List(context.Context, idpenv.ListOptions) ([]idpenv.Resource, error)
	Describe(context.Context, idpenv.DescribeOptions) ([]idpenv.Details, error)
}

type serviceClient interface {
	List(context.Context, idpservice.ListOptions) ([]idpservice.Summary, error)
	Describe(context.Context, idpservice.DescribeOptions) ([]idpservice.Details, error)
}

var newCatalog = func() (catalogClient, error) {
	return catalog.NewKubernetesCatalog()
}

var newCluster = func() (clusterClient, error) {
	return cluster.NewKubernetesCluster()
}

var newEnv = func() (envClient, error) {
	return idpenv.NewKubernetesEnvironment()
}

var newService = func() (serviceClient, error) {
	return idpservice.NewKubernetesService()
}

const helpText = `Hub CLI (IDP)

Usage:
  hub-cli <command> [arguments]

Service commands:
  service list
  service describe <service>
  service health <service>
  service logs <service>
  service events <service>
  service metrics <service>
  service traces <service>
  service ownership <service>

Cluster commands:
  cluster status
  cluster namespaces
  cluster workloads

Catalog commands:
  catalog list
  catalog validate

Environment commands:
  env list
  env describe <name>
`

// Run executes the local IDP CLI command dispatcher.
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, helpText)
		return 0
	}

	switch args[0] {
	case "service":
		return runService(args[1:], stdout, stderr)
	case "cluster":
		return runCluster(args[1:], stdout, stderr)
	case "catalog":
		return runCatalog(args[1:], stdout, stderr)
	case "env":
		return runEnv(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		fmt.Fprint(stderr, helpText)
		return 1
	}
}

func runService(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, serviceHelpText())
		return 0
	}

	switch args[0] {
	case "list":
		opts, ok := parseServiceListOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return listServices(opts, stdout, stderr)
	case "describe":
		if len(args) < 2 {
			fmt.Fprint(stderr, "missing service name for idp service describe\n")
			return 1
		}
		opts, ok := parseServiceDescribeOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return describeService(opts, stdout, stderr)
	case "health", "logs", "events", "metrics", "traces", "ownership":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		return printCalled(stdout, "idp service "+args[0]+" for "+args[1])
	default:
		fmt.Fprintf(stderr, "unknown service command: %s\n\n", args[0])
		fmt.Fprint(stderr, serviceHelpText())
		return 1
	}
}

func runCluster(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, clusterHelpText())
		return 0
	}

	switch args[0] {
	case "status":
		return statusCluster(stdout, stderr)
	case "namespaces":
		return listClusterNamespaces(stdout, stderr)
	case "workloads":
		opts, ok := parseClusterWorkloadOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return listClusterWorkloads(opts, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown cluster command: %s\n\n", args[0])
		fmt.Fprint(stderr, clusterHelpText())
		return 1
	}
}

func runCatalog(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, catalogHelpText())
		return 0
	}

	switch args[0] {
	case "list":
		opts, ok := parseCatalogOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return listCatalog(opts, stdout, stderr)
	case "validate":
		opts, ok := parseCatalogOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return validateCatalog(opts, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown catalog command: %s\n\n", args[0])
		fmt.Fprint(stderr, catalogHelpText())
		return 1
	}
}

func runEnv(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, envHelpText())
		return 0
	}

	switch args[0] {
	case "list":
		opts, ok := parseEnvListOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return listEnvironments(opts, stdout, stderr)
	case "describe":
		if len(args) < 2 {
			fmt.Fprint(stderr, "missing environment name for idp env describe\n")
			return 1
		}
		opts, ok := parseEnvDescribeOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return describeEnvironment(opts, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown env command: %s\n\n", args[0])
		fmt.Fprint(stderr, envHelpText())
		return 1
	}
}

func printCalled(stdout io.Writer, command string) int {
	fmt.Fprintf(stdout, "called %s\n", command)
	return 0
}

func listServices(opts idpservice.ListOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	services, err := serviceClient.List(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tREADY\tAGE")
	for _, service := range services {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", service.Name, service.Namespace, service.Kind, service.Ready, service.Age)
	}
	writer.Flush()
	return 0
}

func describeService(opts idpservice.DescribeOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	services, err := serviceClient.Describe(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	for i, service := range services {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "Name: %s\n", service.Name)
		fmt.Fprintf(stdout, "Namespace: %s\n", service.Namespace)
		fmt.Fprintf(stdout, "Kind: %s\n", service.Kind)
		fmt.Fprintf(stdout, "Ready: %s\n", service.Ready)
		fmt.Fprintf(stdout, "Age: %s\n", service.Age)
		fmt.Fprintln(stdout, "Images:")
		for _, image := range service.Images {
			fmt.Fprintf(stdout, "- %s\n", image)
		}
		fmt.Fprintln(stdout, "Labels:")
		for _, label := range idpservice.SortedMapEntries(service.Labels) {
			fmt.Fprintf(stdout, "- %s\n", label)
		}
		fmt.Fprintln(stdout, "Selectors:")
		for _, selector := range idpservice.SortedMapEntries(service.Selector) {
			fmt.Fprintf(stdout, "- %s\n", selector)
		}
	}
	return 0
}

func statusCluster(stdout io.Writer, stderr io.Writer) int {
	clusterClient, err := newCluster()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	status, err := clusterClient.Status(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "READY_NODES\tTOTAL_NODES\tNAMESPACES\tWORKLOADS")
	fmt.Fprintf(writer, "%d\t%d\t%d\t%d\n", status.ReadyNodes, status.NodeCount, status.NamespaceCount, status.WorkloadCount)
	writer.Flush()
	return 0
}

func listClusterNamespaces(stdout io.Writer, stderr io.Writer) int {
	clusterClient, err := newCluster()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	namespaces, err := clusterClient.Namespaces(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tPHASE\tAGE")
	for _, namespace := range namespaces {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", namespace.Name, namespace.Phase, namespace.Age)
	}
	writer.Flush()
	return 0
}

func listClusterWorkloads(opts cluster.WorkloadOptions, stdout io.Writer, stderr io.Writer) int {
	clusterClient, err := newCluster()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	workloads, err := clusterClient.Workloads(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tREADY\tAGE")
	for _, workload := range workloads {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", workload.Name, workload.Namespace, workload.Kind, workload.Ready, workload.Age)
	}
	writer.Flush()
	return 0
}

func listEnvironments(opts idpenv.ListOptions, stdout io.Writer, stderr io.Writer) int {
	envClient, err := newEnv()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	resources, err := envClient.List(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tTYPE\tKEYS\tAGE")
	for _, resource := range resources {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%d\t%s\n", resource.Name, resource.Namespace, resource.Kind, resource.Type, resource.DataCount, resource.Age)
	}
	writer.Flush()
	return 0
}

func describeEnvironment(opts idpenv.DescribeOptions, stdout io.Writer, stderr io.Writer) int {
	envClient, err := newEnv()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	details, err := envClient.Describe(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	for i, detail := range details {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "Name: %s\n", detail.Name)
		fmt.Fprintf(stdout, "Namespace: %s\n", detail.Namespace)
		fmt.Fprintf(stdout, "Kind: %s\n", detail.Kind)
		fmt.Fprintf(stdout, "Type: %s\n", detail.Type)
		fmt.Fprintf(stdout, "Keys: %d\n", detail.DataCount)
		for _, key := range detail.Keys {
			fmt.Fprintf(stdout, "- %s\n", key)
		}
	}
	return 0
}

func listCatalog(opts catalog.ListOptions, stdout io.Writer, stderr io.Writer) int {
	catalogClient, err := newCatalog()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	entries, err := catalogClient.List(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tSTATUS\tAGE")
	for _, entry := range entries {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", entry.Name, entry.Namespace, entry.Kind, entry.Status, entry.Age)
	}
	writer.Flush()
	return 0
}

func validateCatalog(opts catalog.ListOptions, stdout io.Writer, stderr io.Writer) int {
	catalogClient, err := newCatalog()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	validation, err := catalogClient.Validate(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "catalog valid: discovered %d resources across %d namespaces\n", validation.ResourceCount, validation.NamespaceCount)
	for _, kind := range sortedKinds(validation.KindCounts) {
		fmt.Fprintf(stdout, "- %s: %d\n", kind, validation.KindCounts[kind])
	}
	return 0
}

func parseClusterWorkloadOptions(args []string, stderr io.Writer) (cluster.WorkloadOptions, bool) {
	var opts cluster.WorkloadOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "missing namespace for %s\n", args[i])
				return opts, false
			}
			opts.Namespace = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown cluster workloads option: %s\n", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceListOptions(args []string, stderr io.Writer) (idpservice.ListOptions, bool) {
	var opts idpservice.ListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "missing namespace for %s\n", args[i])
				return opts, false
			}
			opts.Namespace = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown service list option: %s\n", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceDescribeOptions(args []string, stderr io.Writer) (idpservice.DescribeOptions, bool) {
	opts := idpservice.DescribeOptions{Name: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "missing namespace for %s\n", args[i])
				return opts, false
			}
			opts.Namespace = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown service describe option: %s\n", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseCatalogOptions(args []string, stderr io.Writer) (catalog.ListOptions, bool) {
	var opts catalog.ListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "missing namespace for %s\n", args[i])
				return opts, false
			}
			opts.Namespace = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown catalog option: %s\n", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseEnvListOptions(args []string, stderr io.Writer) (idpenv.ListOptions, bool) {
	var opts idpenv.ListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "missing namespace for %s\n", args[i])
				return opts, false
			}
			opts.Namespace = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown env list option: %s\n", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseEnvDescribeOptions(args []string, stderr io.Writer) (idpenv.DescribeOptions, bool) {
	opts := idpenv.DescribeOptions{Name: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "missing namespace for %s\n", args[i])
				return opts, false
			}
			opts.Namespace = args[i+1]
			i++
		case "--kind":
			if i+1 >= len(args) {
				fmt.Fprint(stderr, "missing kind for --kind\n")
				return opts, false
			}
			opts.Kind = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown env describe option: %s\n", args[i])
			return opts, false
		}
	}
	return opts, true
}

func sortedKinds(counts map[string]int) []string {
	kinds := make([]string, 0, len(counts))
	for kind := range counts {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func serviceHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli service list [--namespace <namespace>|-n <namespace>]
  hub-cli service describe <service> [--namespace <namespace>|-n <namespace>]
  hub-cli service health <service>
  hub-cli service logs <service>
  hub-cli service events <service>
  hub-cli service metrics <service>
  hub-cli service traces <service>
  hub-cli service ownership <service>`) + "\n"
}

func clusterHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli cluster status
  hub-cli cluster namespaces
  hub-cli cluster workloads [--namespace <namespace>]`) + "\n"
}

func catalogHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli catalog list [--namespace <namespace>]
  hub-cli catalog validate [--namespace <namespace>]`) + "\n"
}

func envHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli env list [--namespace <namespace>]
  hub-cli env describe <name> [--namespace <namespace>] [--kind <ConfigMap|Secret>]`) + "\n"
}
