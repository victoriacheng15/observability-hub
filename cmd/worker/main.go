package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"observability-hub/internal/env"
	"observability-hub/internal/telemetry"
	"observability-hub/internal/worker"
	"observability-hub/internal/worker/analytics"
	"observability-hub/internal/worker/ingestion"
)

func main() {
	// 1. Parse Flags
	mode := flag.String("mode", "", "Execution mode (analytics or ingestion)")
	flag.Parse()

	if *mode == "" {
		fmt.Println("Error: --mode is required (analytics or ingestion)")
		flag.Usage()
		os.Exit(1)
	}

	// 2. Load Environment
	env.Load()

	// 3. Initialize Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Initialize Telemetry
	serviceName := fmt.Sprintf("worker.%s", *mode)
	shutdown, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		fmt.Printf("Warning: OTel initialization failed: %v\n", err)
	}
	defer shutdown()

	// 5. Initialize Dependencies (Shared for both modes)
	deps, err := worker.InitDependencies(ctx)
	if err != nil {
		telemetry.Error("dependency_init_failed", "error", err)
		os.Exit(1)
	}
	defer deps.Close()

	// 6. Route to specific task
	meter := telemetry.GetMeter("worker.run")
	batchCounter, _ := telemetry.NewInt64Counter(meter, "worker.batch.total", "Total worker runs")
	errorCounter, _ := telemetry.NewInt64Counter(meter, "worker.batch.errors.total", "Total worker errors")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "worker.batch.duration", "Execution time", "ms")
	startTime := time.Now()

	switch *mode {
	case "analytics":
		err = runAnalytics(ctx, deps)
	case "ingestion":
		err = runIngestion(ctx, deps)
	default:
		telemetry.Error("invalid_mode", "mode", *mode)
		os.Exit(1)
	}

	durationMs := time.Since(startTime).Milliseconds()
	telemetry.RecordInt64Histogram(ctx, durationHist, durationMs, telemetry.StringAttribute("mode", *mode))
	telemetry.AddInt64Counter(ctx, batchCounter, 1, telemetry.StringAttribute("mode", *mode))

	if err != nil {
		telemetry.AddInt64Counter(ctx, errorCounter, 1, telemetry.StringAttribute("mode", *mode))
		telemetry.Error("task_execution_failed", "mode", *mode, "error", err)
		os.Exit(1)
	}

	telemetry.Info("worker_task_completed_successfully", "mode", *mode)
}

func runAnalytics(ctx context.Context, deps *worker.Dependencies) error {
	// 1. Setup Analytics specific clients
	thanosClient := analytics.NewThanosClient(deps.GetThanosURL())
	thanosProvider := analytics.NewThanosResourceProvider(thanosClient)

	service := &analytics.Service{
		Store:     deps.Store,
		Resources: thanosProvider,
	}

	// 2. Execute one-shot batch
	return service.Run(ctx)
}

func runIngestion(ctx context.Context, deps *worker.Dependencies) error {
	if deps.Store == nil {
		return fmt.Errorf("database connection is required for ingestion")
	}

	// 1. Setup Ingestion App
	app := ingestion.NewApp(deps.SecretStore, deps.Store)

	// 2. Execute all tasks
	return app.Run(ctx)
}
