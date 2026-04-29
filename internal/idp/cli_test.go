package idp

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"observability-hub/internal/idp/catalog"
	"observability-hub/internal/idp/cluster"
	idpenv "observability-hub/internal/idp/env"
	idpservice "observability-hub/internal/idp/service"
)

func TestRunPlaceholderCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if got := stdout.String(); !strings.Contains(got, "service list") {
		t.Fatalf("help output missing service command: %q", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRunServiceCommands(t *testing.T) {
	fake := &fakeService{
		summaries: []idpservice.Summary{
			{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/1", Age: "2h"},
			{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Ready: "1/1", Age: "2h"},
		},
		details: []idpservice.Details{
			{
				Summary:  idpservice.Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/1", Age: "2h"},
				Images:   []string{"grafana/grafana:latest"},
				Labels:   map[string]string{"app.kubernetes.io/name": "grafana", "tier": "frontend"},
				Selector: map[string]string{"app.kubernetes.io/name": "grafana"},
			},
		},
		health: []idpservice.Health{
			{
				Summary:           idpservice.Summary{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/1", Age: "2h"},
				Status:            "healthy",
				ReadyPods:         2,
				PodCount:          2,
				ReadyServices:     1,
				ServiceCount:      1,
				WarningEventCount: 0,
			},
		},
		logs: []idpservice.LogLine{
			{Name: "grafana", Namespace: "observability", Kind: "Deployment", Pod: "grafana-0", Line: "ready"},
		},
		events: []idpservice.Event{
			{Type: "Warning", Reason: "BackOff", Age: "1m", Object: "Pod/grafana-0", Message: "back-off restarting failed container"},
		},
		metrics: []idpservice.Metric{
			{Name: "grafana", Namespace: "observability", Kind: "Deployment", Signal: "cpu_cores", Value: "0.12"},
		},
		traces: []idpservice.Trace{
			{Name: "grafana", Namespace: "observability", Kind: "Deployment", TraceID: "abc123", RootServiceName: "grafana", StartTime: "2026-04-28T12:00:00Z", Duration: "25ms"},
		},
	}
	restore := stubService(t, fake)
	defer restore()

	tests := []struct {
		name     string
		args     []string
		want     string
		wantList []idpservice.ListOptions
		wantDesc []idpservice.DescribeOptions
		wantHeal []idpservice.HealthOptions
		wantLogs []idpservice.LogsOptions
		wantEvts []idpservice.EventsOptions
		wantMets []idpservice.MetricsOptions
		wantTrcs []idpservice.TracesOptions
	}{
		{
			name: "service list",
			args: []string{"service", "list"},
			want: "NAME     NAMESPACE      KIND         READY  AGE\n" +
				"grafana  observability  Deployment   1/1    2h\n" +
				"loki     observability  StatefulSet  1/1    2h\n",
			wantList: []idpservice.ListOptions{{}},
		},
		{
			name: "service list namespace",
			args: []string{"service", "list", "-n", "observability"},
			want: "NAME     NAMESPACE      KIND         READY  AGE\n" +
				"grafana  observability  Deployment   1/1    2h\n" +
				"loki     observability  StatefulSet  1/1    2h\n",
			wantList: []idpservice.ListOptions{{Namespace: "observability"}},
		},
		{
			name: "service describe",
			args: []string{"service", "describe", "grafana", "--namespace", "observability"},
			want: "Name: grafana\n" +
				"Namespace: observability\n" +
				"Kind: Deployment\n" +
				"Ready: 1/1\n" +
				"Age: 2h\n" +
				"Images:\n" +
				"- grafana/grafana:latest\n" +
				"Labels:\n" +
				"- app.kubernetes.io/name=grafana\n" +
				"- tier=frontend\n" +
				"Selectors:\n" +
				"- app.kubernetes.io/name=grafana\n",
			wantDesc: []idpservice.DescribeOptions{{Name: "grafana", Namespace: "observability"}},
		},
		{
			name: "service health",
			args: []string{"service", "health", "grafana", "-n", "observability"},
			want: "NAME     NAMESPACE      KIND        STATUS   READY  PODS  RESTARTS  SERVICES  WARNINGS\n" +
				"grafana  observability  Deployment  healthy  1/1    2/2   0         1/1       0\n",
			wantHeal: []idpservice.HealthOptions{{Name: "grafana", Namespace: "observability"}},
		},
		{
			name:     "service logs",
			args:     []string{"service", "logs", "grafana", "-n", "observability", "--container", "app", "--tail", "10", "--since", "5m", "--previous"},
			want:     "observability/grafana-0 grafana ready\n",
			wantLogs: []idpservice.LogsOptions{{Name: "grafana", Namespace: "observability", Container: "app", Tail: 10, Since: 5 * time.Minute, Previous: true}},
		},
		{
			name: "service events",
			args: []string{"service", "events", "grafana", "-n", "observability", "--tail", "5", "--since", "10m"},
			want: "TYPE     REASON   AGE  OBJECT         MESSAGE\n" +
				"Warning  BackOff  1m   Pod/grafana-0  back-off restarting failed container\n",
			wantEvts: []idpservice.EventsOptions{{Name: "grafana", Namespace: "observability", Tail: 5, Since: 10 * time.Minute}},
		},
		{
			name: "service metrics",
			args: []string{"service", "metrics", "grafana", "-n", "observability", "--window", "10m"},
			want: "NAME     NAMESPACE      KIND        SIGNAL     VALUE\n" +
				"grafana  observability  Deployment  cpu_cores  0.12\n",
			wantMets: []idpservice.MetricsOptions{{Name: "grafana", Namespace: "observability", Window: 10 * time.Minute}},
		},
		{
			name: "service traces",
			args: []string{"service", "traces", "grafana", "-n", "observability", "--hours", "2", "--limit", "5"},
			want: "NAME     NAMESPACE      KIND        TRACE_ID  ROOT_SERVICE  START                 DURATION\n" +
				"grafana  observability  Deployment  abc123    grafana       2026-04-28T12:00:00Z  25ms\n",
			wantTrcs: []idpservice.TracesOptions{{Name: "grafana", Namespace: "observability", Hours: 2, Limit: 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.listOpts = nil
			fake.describeOpts = nil
			fake.healthOpts = nil
			fake.logsOpts = nil
			fake.eventsOpts = nil
			fake.metricsOpts = nil
			fake.tracesOpts = nil
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
			assertServiceListOptions(t, fake.listOpts, tt.wantList)
			assertServiceDescribeOptions(t, fake.describeOpts, tt.wantDesc)
			assertServiceHealthOptions(t, fake.healthOpts, tt.wantHeal)
			assertServiceLogsOptions(t, fake.logsOpts, tt.wantLogs)
			assertServiceEventsOptions(t, fake.eventsOpts, tt.wantEvts)
			assertServiceMetricsOptions(t, fake.metricsOpts, tt.wantMets)
			assertServiceTracesOptions(t, fake.tracesOpts, tt.wantTrcs)
		})
	}
}

func TestRunServiceEventsEmpty(t *testing.T) {
	fake := &fakeService{}
	restore := stubService(t, fake)
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"service", "events", "grafana"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != "no events found for grafana\n" {
		t.Fatalf("stdout = %q, want empty event message", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
	assertServiceEventsOptions(t, fake.eventsOpts, []idpservice.EventsOptions{{Name: "grafana", Tail: 25}})
}

func TestRunServiceOptionErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing namespace",
			args: []string{"service", "list", "--namespace"},
			want: "missing namespace for --namespace",
		},
		{
			name: "unknown list option",
			args: []string{"service", "list", "--team", "payments"},
			want: "unknown service list option: --team",
		},
		{
			name: "unknown describe option",
			args: []string{"service", "describe", "grafana", "--team", "payments"},
			want: "unknown service describe option: --team",
		},
		{
			name: "unknown health option",
			args: []string{"service", "health", "grafana", "--team", "payments"},
			want: "unknown service health option: --team",
		},
		{
			name: "invalid logs tail",
			args: []string{"service", "logs", "grafana", "--tail", "zero"},
			want: "invalid tail for --tail: zero",
		},
		{
			name: "unknown events option",
			args: []string{"service", "events", "grafana", "--container", "app"},
			want: "unknown service events option: --container",
		},
		{
			name: "invalid metrics window",
			args: []string{"service", "metrics", "grafana", "--window", "soon"},
			want: "invalid window for --window: soon",
		},
		{
			name: "unknown traces option",
			args: []string{"service", "traces", "grafana", "--window", "5m"},
			want: "unknown service traces option: --window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("Run() code = %d, want 1", code)
			}
			if got := stdout.String(); got != "" {
				t.Fatalf("stdout = %q, want empty", got)
			}
			if got := stderr.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("stderr = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestRunClusterCommands(t *testing.T) {
	fake := &fakeCluster{
		status: cluster.Status{
			ReadyNodes:     1,
			NodeCount:      2,
			NamespaceCount: 3,
			WorkloadCount:  4,
		},
		namespaces: []cluster.Namespace{
			{Name: "default", Phase: "Active", Age: "3d"},
			{Name: "observability", Phase: "Active", Age: "2d"},
		},
		workloads: []cluster.Workload{
			{Name: "grafana", Namespace: "observability", Kind: "Deployment", Ready: "1/1", Age: "2h"},
			{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Ready: "1/1", Age: "2h"},
		},
	}
	restore := stubCluster(t, fake)
	defer restore()

	tests := []struct {
		name     string
		args     []string
		want     string
		wantWork []cluster.WorkloadOptions
	}{
		{
			name: "cluster status",
			args: []string{"cluster", "status"},
			want: "READY_NODES  TOTAL_NODES  NAMESPACES  WORKLOADS\n" +
				"1            2            3           4\n",
		},
		{
			name: "cluster namespaces",
			args: []string{"cluster", "namespaces"},
			want: "NAME           PHASE   AGE\n" +
				"default        Active  3d\n" +
				"observability  Active  2d\n",
		},
		{
			name: "cluster workloads",
			args: []string{"cluster", "workloads"},
			want: "NAME     NAMESPACE      KIND         READY  AGE\n" +
				"grafana  observability  Deployment   1/1    2h\n" +
				"loki     observability  StatefulSet  1/1    2h\n",
			wantWork: []cluster.WorkloadOptions{{}},
		},
		{
			name: "cluster workloads namespace",
			args: []string{"cluster", "workloads", "-n", "payments"},
			want: "NAME     NAMESPACE      KIND         READY  AGE\n" +
				"grafana  observability  Deployment   1/1    2h\n" +
				"loki     observability  StatefulSet  1/1    2h\n",
			wantWork: []cluster.WorkloadOptions{{Namespace: "payments"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.workloadOpts = nil
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
			assertWorkloadOptions(t, fake.workloadOpts, tt.wantWork)
		})
	}
}

func TestRunClusterWorkloadOptionErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing namespace",
			args: []string{"cluster", "workloads", "--namespace"},
			want: "missing namespace for --namespace",
		},
		{
			name: "unknown option",
			args: []string{"cluster", "workloads", "--team", "payments"},
			want: "unknown cluster workloads option: --team",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("Run() code = %d, want 1", code)
			}
			if got := stdout.String(); got != "" {
				t.Fatalf("stdout = %q, want empty", got)
			}
			if got := stderr.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("stderr = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestRunCatalogCommands(t *testing.T) {
	fake := &fakeCatalog{
		entries: []catalog.Entry{
			{Name: "grafana", Namespace: "observability", Kind: "Deployment", Status: "1/1", Age: "2h"},
			{Name: "loki", Namespace: "observability", Kind: "StatefulSet", Status: "1/1", Age: "2h"},
		},
		validation: catalog.Validation{
			ResourceCount:  2,
			NamespaceCount: 1,
			KindCounts: map[string]int{
				"Deployment":  1,
				"StatefulSet": 1,
			},
		},
	}
	restore := stubCatalog(t, fake)
	defer restore()

	tests := []struct {
		name       string
		args       []string
		want       string
		wantList   []catalog.ListOptions
		wantValid  []catalog.ListOptions
		resetCalls bool
	}{
		{
			name: "catalog list",
			args: []string{"catalog", "list"},
			want: "NAME     NAMESPACE      KIND         STATUS  AGE\n" +
				"grafana  observability  Deployment   1/1     2h\n" +
				"loki     observability  StatefulSet  1/1     2h\n",
			wantList: []catalog.ListOptions{{}},
		},
		{
			name:      "catalog validate",
			args:      []string{"catalog", "validate"},
			want:      "catalog valid: discovered 2 resources across 1 namespaces\n- Deployment: 1\n- StatefulSet: 1\n",
			wantValid: []catalog.ListOptions{{}},
		},
		{
			name:     "catalog list namespace",
			args:     []string{"catalog", "list", "--namespace", "payments"},
			want:     "NAME     NAMESPACE      KIND         STATUS  AGE\n" + "grafana  observability  Deployment   1/1     2h\n" + "loki     observability  StatefulSet  1/1     2h\n",
			wantList: []catalog.ListOptions{{Namespace: "payments"}},
		},
		{
			name:      "catalog validate shorthand namespace",
			args:      []string{"catalog", "validate", "-n", "payments"},
			want:      "catalog valid: discovered 2 resources across 1 namespaces\n- Deployment: 1\n- StatefulSet: 1\n",
			wantValid: []catalog.ListOptions{{Namespace: "payments"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.listOpts = nil
			fake.validOpts = nil
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
			assertListOptions(t, fake.listOpts, tt.wantList)
			assertListOptions(t, fake.validOpts, tt.wantValid)
		})
	}
}

func TestRunCatalogNamespaceOptionErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing namespace",
			args: []string{"catalog", "list", "--namespace"},
			want: "missing namespace for --namespace",
		},
		{
			name: "unknown option",
			args: []string{"catalog", "validate", "--team", "payments"},
			want: "unknown catalog option: --team",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("Run() code = %d, want 1", code)
			}
			if got := stdout.String(); got != "" {
				t.Fatalf("stdout = %q, want empty", got)
			}
			if got := stderr.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("stderr = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestRunEnvCommands(t *testing.T) {
	fake := &fakeEnv{
		resources: []idpenv.Resource{
			{Name: "grafana-env", Namespace: "observability", Kind: "ConfigMap", Type: "-", DataCount: 2, Age: "2h"},
			{Name: "grafana-admin", Namespace: "observability", Kind: "Secret", Type: "Opaque", DataCount: 2, Age: "2h"},
		},
		details: []idpenv.Details{
			{
				Resource: idpenv.Resource{Name: "grafana-admin", Namespace: "observability", Kind: "Secret", Type: "Opaque", DataCount: 2, Age: "2h"},
				Keys:     []string{"password", "username"},
			},
		},
	}
	restore := stubEnv(t, fake)
	defer restore()

	tests := []struct {
		name     string
		args     []string
		want     string
		wantList []idpenv.ListOptions
		wantDesc []idpenv.DescribeOptions
	}{
		{
			name: "env list",
			args: []string{"env", "list"},
			want: "NAME           NAMESPACE      KIND       TYPE    KEYS  AGE\n" +
				"grafana-env    observability  ConfigMap  -       2     2h\n" +
				"grafana-admin  observability  Secret     Opaque  2     2h\n",
			wantList: []idpenv.ListOptions{{}},
		},
		{
			name: "env list namespace",
			args: []string{"env", "list", "--namespace", "observability"},
			want: "NAME           NAMESPACE      KIND       TYPE    KEYS  AGE\n" +
				"grafana-env    observability  ConfigMap  -       2     2h\n" +
				"grafana-admin  observability  Secret     Opaque  2     2h\n",
			wantList: []idpenv.ListOptions{{Namespace: "observability"}},
		},
		{
			name: "env describe",
			args: []string{"env", "describe", "grafana-admin", "-n", "observability", "--kind", "Secret"},
			want: "Name: grafana-admin\n" +
				"Namespace: observability\n" +
				"Kind: Secret\n" +
				"Type: Opaque\n" +
				"Keys: 2\n" +
				"- password\n" +
				"- username\n",
			wantDesc: []idpenv.DescribeOptions{{Name: "grafana-admin", Namespace: "observability", Kind: "Secret"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.listOpts = nil
			fake.describeOpts = nil
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
			assertEnvListOptions(t, fake.listOpts, tt.wantList)
			assertEnvDescribeOptions(t, fake.describeOpts, tt.wantDesc)
		})
	}
}

func TestRunEnvOptionErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing namespace",
			args: []string{"env", "list", "--namespace"},
			want: "missing namespace for --namespace",
		},
		{
			name: "unknown list option",
			args: []string{"env", "list", "--team", "payments"},
			want: "unknown env list option: --team",
		},
		{
			name: "missing kind",
			args: []string{"env", "describe", "grafana-admin", "--kind"},
			want: "missing kind for --kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("Run() code = %d, want 1", code)
			}
			if got := stdout.String(); got != "" {
				t.Fatalf("stdout = %q, want empty", got)
			}
			if got := stderr.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("stderr = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestRunMissingRequiredArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"service", "describe"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); !strings.Contains(got, "missing service name") {
		t.Fatalf("stderr missing required argument message: %q", got)
	}
}

type fakeEnv struct {
	resources    []idpenv.Resource
	details      []idpenv.Details
	listOpts     []idpenv.ListOptions
	describeOpts []idpenv.DescribeOptions
}

func (f *fakeEnv) List(_ context.Context, opts idpenv.ListOptions) ([]idpenv.Resource, error) {
	f.listOpts = append(f.listOpts, opts)
	return f.resources, nil
}

func (f *fakeEnv) Describe(_ context.Context, opts idpenv.DescribeOptions) ([]idpenv.Details, error) {
	f.describeOpts = append(f.describeOpts, opts)
	return f.details, nil
}

type fakeService struct {
	summaries    []idpservice.Summary
	details      []idpservice.Details
	health       []idpservice.Health
	logs         []idpservice.LogLine
	events       []idpservice.Event
	metrics      []idpservice.Metric
	traces       []idpservice.Trace
	listOpts     []idpservice.ListOptions
	describeOpts []idpservice.DescribeOptions
	healthOpts   []idpservice.HealthOptions
	logsOpts     []idpservice.LogsOptions
	eventsOpts   []idpservice.EventsOptions
	metricsOpts  []idpservice.MetricsOptions
	tracesOpts   []idpservice.TracesOptions
}

func (f *fakeService) List(_ context.Context, opts idpservice.ListOptions) ([]idpservice.Summary, error) {
	f.listOpts = append(f.listOpts, opts)
	return f.summaries, nil
}

func (f *fakeService) Describe(_ context.Context, opts idpservice.DescribeOptions) ([]idpservice.Details, error) {
	f.describeOpts = append(f.describeOpts, opts)
	return f.details, nil
}

func (f *fakeService) Health(_ context.Context, opts idpservice.HealthOptions) ([]idpservice.Health, error) {
	f.healthOpts = append(f.healthOpts, opts)
	return f.health, nil
}

func (f *fakeService) Logs(_ context.Context, opts idpservice.LogsOptions) ([]idpservice.LogLine, error) {
	f.logsOpts = append(f.logsOpts, opts)
	return f.logs, nil
}

func (f *fakeService) Events(_ context.Context, opts idpservice.EventsOptions) ([]idpservice.Event, error) {
	f.eventsOpts = append(f.eventsOpts, opts)
	return f.events, nil
}

func (f *fakeService) Metrics(_ context.Context, opts idpservice.MetricsOptions) ([]idpservice.Metric, error) {
	f.metricsOpts = append(f.metricsOpts, opts)
	return f.metrics, nil
}

func (f *fakeService) Traces(_ context.Context, opts idpservice.TracesOptions) ([]idpservice.Trace, error) {
	f.tracesOpts = append(f.tracesOpts, opts)
	return f.traces, nil
}

type fakeCluster struct {
	status       cluster.Status
	namespaces   []cluster.Namespace
	workloads    []cluster.Workload
	workloadOpts []cluster.WorkloadOptions
}

func (f *fakeCluster) Status(_ context.Context) (cluster.Status, error) {
	return f.status, nil
}

func (f *fakeCluster) Namespaces(_ context.Context) ([]cluster.Namespace, error) {
	return f.namespaces, nil
}

func (f *fakeCluster) Workloads(_ context.Context, opts cluster.WorkloadOptions) ([]cluster.Workload, error) {
	f.workloadOpts = append(f.workloadOpts, opts)
	return f.workloads, nil
}

type fakeCatalog struct {
	entries    []catalog.Entry
	validation catalog.Validation
	listOpts   []catalog.ListOptions
	validOpts  []catalog.ListOptions
}

func (f *fakeCatalog) List(_ context.Context, opts catalog.ListOptions) ([]catalog.Entry, error) {
	f.listOpts = append(f.listOpts, opts)
	return f.entries, nil
}

func (f *fakeCatalog) Validate(_ context.Context, opts catalog.ListOptions) (catalog.Validation, error) {
	f.validOpts = append(f.validOpts, opts)
	return f.validation, nil
}

func assertListOptions(t *testing.T, got []catalog.ListOptions, want []catalog.ListOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertWorkloadOptions(t *testing.T, got []cluster.WorkloadOptions, want []cluster.WorkloadOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertEnvListOptions(t *testing.T, got []idpenv.ListOptions, want []idpenv.ListOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertEnvDescribeOptions(t *testing.T, got []idpenv.DescribeOptions, want []idpenv.DescribeOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceListOptions(t *testing.T, got []idpservice.ListOptions, want []idpservice.ListOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceDescribeOptions(t *testing.T, got []idpservice.DescribeOptions, want []idpservice.DescribeOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceHealthOptions(t *testing.T, got []idpservice.HealthOptions, want []idpservice.HealthOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceLogsOptions(t *testing.T, got []idpservice.LogsOptions, want []idpservice.LogsOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceEventsOptions(t *testing.T, got []idpservice.EventsOptions, want []idpservice.EventsOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceMetricsOptions(t *testing.T, got []idpservice.MetricsOptions, want []idpservice.MetricsOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertServiceTracesOptions(t *testing.T, got []idpservice.TracesOptions, want []idpservice.TracesOptions) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(options) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("options[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func stubService(t *testing.T, fake *fakeService) func() {
	t.Helper()

	original := newService
	newService = func() (serviceClient, error) {
		return fake, nil
	}
	return func() {
		newService = original
	}
}

func stubEnv(t *testing.T, fake *fakeEnv) func() {
	t.Helper()

	original := newEnv
	newEnv = func() (envClient, error) {
		return fake, nil
	}
	return func() {
		newEnv = original
	}
}

func stubCluster(t *testing.T, fake *fakeCluster) func() {
	t.Helper()

	original := newCluster
	newCluster = func() (clusterClient, error) {
		return fake, nil
	}
	return func() {
		newCluster = original
	}
}

func stubCatalog(t *testing.T, fake *fakeCatalog) func() {
	t.Helper()

	original := newCatalog
	newCatalog = func() (catalogClient, error) {
		return fake, nil
	}
	return func() {
		newCatalog = original
	}
}
