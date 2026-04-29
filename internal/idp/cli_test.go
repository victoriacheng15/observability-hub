package idp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"observability-hub/internal/idp/catalog"
)

func TestRunPlaceholderCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "service list",
			args: []string{"service", "list"},
			want: "called idp service list\n",
		},
		{
			name: "service describe",
			args: []string{"service", "describe", "loki"},
			want: "called idp service describe for loki\n",
		},
		{
			name: "service health",
			args: []string{"service", "health", "tempo"},
			want: "called idp service health for tempo\n",
		},
		{
			name: "cluster status",
			args: []string{"cluster", "status"},
			want: "called idp cluster status\n",
		},
		{
			name: "env describe",
			args: []string{"env", "describe", "local"},
			want: "called idp env describe for local\n",
		},
	}

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
