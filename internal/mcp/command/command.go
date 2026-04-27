package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"observability-hub/internal/telemetry"
)

const DefaultTimeout = 10 * time.Second

// Request describes one external command invocation.
type Request struct {
	Name    string
	Args    []string
	Stdin   []byte
	Timeout time.Duration
}

// Result contains captured process output and execution metadata.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	Duration time.Duration
}

// Runner executes external commands.
type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}

// Executor is the production command boundary for MCP tools.
type Executor struct {
	allowed        map[string]struct{}
	defaultTimeout time.Duration
}

// NewExecutor creates an Executor with an explicit binary allowlist.
func NewExecutor(allowed []string) *Executor {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}
	return &Executor{
		allowed:        allowedSet,
		defaultTimeout: DefaultTimeout,
	}
}

// Run executes the requested command with timeout, captured output, and typed errors.
func (e *Executor) Run(ctx context.Context, req Request) (Result, error) {
	if req.Name == "" {
		return Result{}, fmt.Errorf("command name is required")
	}
	if _, ok := e.allowed[req.Name]; !ok {
		return Result{}, fmt.Errorf("command %q is not allowed", req.Name)
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}

	childCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(childCtx, req.Name, req.Args...)
	if req.Stdin != nil {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)
	result := Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Duration: duration,
	}
	if err == nil {
		telemetry.Info("mcp command executed", "command", req.Name, "duration_ms", duration.Milliseconds())
		return result, nil
	}

	commandErr := &Error{
		Name:     req.Name,
		Args:     append([]string(nil), req.Args...),
		Stderr:   stderr.String(),
		Err:      err,
		Duration: duration,
		TimedOut: childCtx.Err() == context.DeadlineExceeded,
	}
	telemetry.Error(
		"mcp command failed",
		"command", req.Name,
		"duration_ms", duration.Milliseconds(),
		"timed_out", commandErr.TimedOut,
		"error", err,
	)
	return result, commandErr
}

// Error is a structured external command failure.
type Error struct {
	Name     string
	Args     []string
	Stderr   string
	Err      error
	Duration time.Duration
	TimedOut bool
}

func (e *Error) Error() string {
	if e.TimedOut {
		return fmt.Sprintf("command %q timed out after %s: %v", e.Name, e.Duration.Round(time.Millisecond), e.Err)
	}
	if e.Stderr != "" {
		return fmt.Sprintf("command %q failed: %v: %s", e.Name, e.Err, e.Stderr)
	}
	return fmt.Sprintf("command %q failed: %v", e.Name, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
