package command

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecutor_RejectsDisallowedCommand(t *testing.T) {
	executor := NewExecutor([]string{"kubectl"})

	_, err := executor.Run(context.Background(), Request{Name: "sh"})
	if err == nil {
		t.Fatal("expected disallowed command error")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("got %q, want not allowed error", err.Error())
	}
}

func TestError_ErrorIncludesTimeout(t *testing.T) {
	err := &Error{
		Name:     "kubectl",
		Err:      context.DeadlineExceeded,
		Duration: 10 * time.Millisecond,
		TimedOut: true,
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("got %q, want timeout context", err.Error())
	}
}

func TestError_ErrorIncludesStderr(t *testing.T) {
	err := &Error{
		Name:   "kubectl",
		Err:    context.Canceled,
		Stderr: "permission denied",
	}

	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("got %q, want stderr context", err.Error())
	}
}
