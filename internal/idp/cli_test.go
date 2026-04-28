package idp

import (
	"bytes"
	"strings"
	"testing"
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
			name: "catalog validate",
			args: []string{"catalog", "validate"},
			want: "called idp catalog validate\n",
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
