package idp

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

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
	Health(context.Context, idpservice.HealthOptions) ([]idpservice.Health, error)
	Logs(context.Context, idpservice.LogsOptions) ([]idpservice.LogLine, error)
	Events(context.Context, idpservice.EventsOptions) ([]idpservice.Event, error)
	Metrics(context.Context, idpservice.MetricsOptions) ([]idpservice.Metric, error)
	Traces(context.Context, idpservice.TracesOptions) ([]idpservice.Trace, error)
	Ownership(context.Context, idpservice.OwnershipOptions) ([]idpservice.Ownership, error)
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
	case "health":
		if len(args) < 2 {
			fmt.Fprint(stderr, "missing service name for idp service health\n")
			return 1
		}
		opts, ok := parseServiceHealthOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return healthService(opts, stdout, stderr)
	case "logs":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		opts, ok := parseServiceLogsOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return logsService(opts, stdout, stderr)
	case "events":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		opts, ok := parseServiceEventsOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return eventsService(opts, stdout, stderr)
	case "metrics":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		opts, ok := parseServiceMetricsOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return metricsService(opts, stdout, stderr)
	case "traces":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		opts, ok := parseServiceTracesOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return tracesService(opts, stdout, stderr)
	case "ownership":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		opts, ok := parseServiceOwnershipOptions(args[1:], stderr)
		if !ok {
			return 1
		}
		return ownershipService(opts, stdout, stderr)
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

func healthService(opts idpservice.HealthOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	health, err := serviceClient.Health(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tSTATUS\tREADY\tPODS\tRESTARTS\tSERVICES\tWARNINGS")
	for _, item := range health {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%d/%d\t%d\t%d/%d\t%d\n",
			item.Name,
			item.Namespace,
			item.Kind,
			item.Status,
			item.Ready,
			item.ReadyPods,
			item.PodCount,
			item.Restarts,
			item.ReadyServices,
			item.ServiceCount,
			item.WarningEventCount,
		)
	}
	writer.Flush()

	for _, item := range health {
		if len(item.WarningEventReasons) == 0 {
			continue
		}
		fmt.Fprintf(stdout, "%s/%s warning events:\n", item.Namespace, item.Name)
		for _, reason := range item.WarningEventReasons {
			fmt.Fprintf(stdout, "- %s\n", reason)
		}
	}

	return 0
}

func logsService(opts idpservice.LogsOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	logs, err := serviceClient.Logs(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if len(logs) == 0 {
		fmt.Fprintf(stdout, "no logs found for %s\n", opts.Name)
		return 0
	}

	for _, line := range logs {
		fmt.Fprintf(stdout, "%s/%s %s %s\n", line.Namespace, line.Pod, line.Name, line.Line)
	}
	return 0
}

func eventsService(opts idpservice.EventsOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	events, err := serviceClient.Events(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if len(events) == 0 {
		fmt.Fprintf(stdout, "no events found for %s\n", opts.Name)
		return 0
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TYPE\tREASON\tAGE\tOBJECT\tMESSAGE")
	for _, event := range events {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", event.Type, event.Reason, event.Age, event.Object, event.Message)
	}
	writer.Flush()
	return 0
}

func metricsService(opts idpservice.MetricsOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	metrics, err := serviceClient.Metrics(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if len(metrics) == 0 {
		fmt.Fprintf(stdout, "no metrics found for %s\n", opts.Name)
		return 0
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tSIGNAL\tVALUE")
	for _, metric := range metrics {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", metric.Name, metric.Namespace, metric.Kind, metric.Signal, metric.Value)
	}
	writer.Flush()
	return 0
}

func tracesService(opts idpservice.TracesOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	traces, err := serviceClient.Traces(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if len(traces) == 0 {
		fmt.Fprintf(stdout, "no traces found for %s\n", opts.Name)
		return 0
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tTRACE_ID\tROOT_SERVICE\tSTART\tDURATION")
	for _, trace := range traces {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", trace.Name, trace.Namespace, trace.Kind, trace.TraceID, trace.RootServiceName, trace.StartTime, trace.Duration)
	}
	writer.Flush()
	return 0
}

func ownershipService(opts idpservice.OwnershipOptions, stdout io.Writer, stderr io.Writer) int {
	serviceClient, err := newService()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	ownership, err := serviceClient.Ownership(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tNAMESPACE\tKIND\tOWNER\tTIER\tSOURCE\tDOCS\tSOURCE_OBJECT")
	for _, item := range ownership {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", item.Name, item.Namespace, item.Kind, item.Owner, item.Tier, item.Source, strings.Join(item.Docs, ","), item.SourceObject)
	}
	writer.Flush()
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
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "cluster workloads", args[i])
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
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "service list", args[i])
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
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "service describe", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceHealthOptions(args []string, stderr io.Writer) (idpservice.HealthOptions, bool) {
	opts := idpservice.HealthOptions{Name: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "service health", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceLogsOptions(args []string, stderr io.Writer) (idpservice.LogsOptions, bool) {
	opts := idpservice.LogsOptions{Name: args[0], Tail: 100}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		case "--container", "-c":
			container, ok := requireValue(args, i, "container", stderr)
			if !ok {
				return opts, false
			}
			opts.Container = container
			i++
		case "--tail":
			tail, ok := parsePositiveInt64(args, i, "tail", stderr)
			if !ok {
				return opts, false
			}
			opts.Tail = tail
			i++
		case "--since":
			since, ok := parseDurationOption(args, i, "since", stderr)
			if !ok {
				return opts, false
			}
			opts.Since = since
			i++
		case "--previous":
			opts.Previous = true
		default:
			unknownOption(stderr, "service logs", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceEventsOptions(args []string, stderr io.Writer) (idpservice.EventsOptions, bool) {
	opts := idpservice.EventsOptions{Name: args[0], Tail: 25}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		case "--tail":
			tail, ok := parsePositiveInt(args, i, "tail", stderr)
			if !ok {
				return opts, false
			}
			opts.Tail = tail
			i++
		case "--since":
			since, ok := parseDurationOption(args, i, "since", stderr)
			if !ok {
				return opts, false
			}
			opts.Since = since
			i++
		default:
			unknownOption(stderr, "service events", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceMetricsOptions(args []string, stderr io.Writer) (idpservice.MetricsOptions, bool) {
	opts := idpservice.MetricsOptions{Name: args[0], Window: 5 * time.Minute}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		case "--window":
			window, ok := parseDurationOption(args, i, "window", stderr)
			if !ok {
				return opts, false
			}
			opts.Window = window
			i++
		default:
			unknownOption(stderr, "service metrics", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceTracesOptions(args []string, stderr io.Writer) (idpservice.TracesOptions, bool) {
	opts := idpservice.TracesOptions{Name: args[0], Hours: 1, Limit: 20}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		case "--hours":
			hours, ok := parsePositiveInt(args, i, "hours", stderr)
			if !ok {
				return opts, false
			}
			opts.Hours = hours
			i++
		case "--limit":
			limit, ok := parsePositiveInt(args, i, "limit", stderr)
			if !ok {
				return opts, false
			}
			opts.Limit = limit
			i++
		default:
			unknownOption(stderr, "service traces", args[i])
			return opts, false
		}
	}
	return opts, true
}

func parseServiceOwnershipOptions(args []string, stderr io.Writer) (idpservice.OwnershipOptions, bool) {
	opts := idpservice.OwnershipOptions{Name: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--namespace", "-n":
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "service ownership", args[i])
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
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "catalog", args[i])
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
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		default:
			unknownOption(stderr, "env list", args[i])
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
			namespace, ok := parseNamespaceOption(args, i, stderr)
			if !ok {
				return opts, false
			}
			opts.Namespace = namespace
			i++
		case "--kind":
			kind, ok := requireValue(args, i, "kind", stderr)
			if !ok {
				return opts, false
			}
			opts.Kind = kind
			i++
		default:
			unknownOption(stderr, "env describe", args[i])
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

func parsePositiveInt(args []string, index int, name string, stderr io.Writer) (int, bool) {
	value, ok := parsePositiveInt64(args, index, name, stderr)
	return int(value), ok
}

func parsePositiveInt64(args []string, index int, name string, stderr io.Writer) (int64, bool) {
	raw, ok := requireValue(args, index, name, stderr)
	if !ok {
		return 0, false
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		fmt.Fprintf(stderr, "invalid %s for %s: %s\n", name, args[index], raw)
		return 0, false
	}
	return value, true
}

func parseDurationOption(args []string, index int, name string, stderr io.Writer) (time.Duration, bool) {
	raw, ok := requireValue(args, index, name, stderr)
	if !ok {
		return 0, false
	}

	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		fmt.Fprintf(stderr, "invalid %s for %s: %s\n", name, args[index], raw)
		return 0, false
	}
	return value, true
}

func parseNamespaceOption(args []string, index int, stderr io.Writer) (string, bool) {
	return requireValue(args, index, "namespace", stderr)
}

func requireValue(args []string, index int, name string, stderr io.Writer) (string, bool) {
	if index+1 >= len(args) {
		fmt.Fprintf(stderr, "missing %s for %s\n", name, args[index])
		return "", false
	}
	return args[index+1], true
}

func unknownOption(stderr io.Writer, command string, option string) {
	fmt.Fprintf(stderr, "unknown %s option: %s\n", command, option)
}
