package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"observability-hub/internal/env"
	"observability-hub/internal/telemetry"
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

	// 4. Initialize Telemetry with dynamic service name
	serviceName := fmt.Sprintf("worker.%s", *mode)
	shutdown, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		fmt.Printf("Warning: OTel initialization failed: %v\n", err)
	}
	defer shutdown()

	// 5. Route to specific task
	switch *mode {
	case "analytics":
		runAnalytics(ctx)
	case "ingestion":
		runIngestion(ctx)
	default:
		telemetry.Error("invalid_mode", "mode", *mode)
		os.Exit(1)
	}
}

func runAnalytics(ctx context.Context) {
	telemetry.Info("starting_no_op_analytics")
	// Placeholder for PR 2
	telemetry.Info("analytics_task_finished")
}

func runIngestion(ctx context.Context) {
	telemetry.Info("starting_no_op_ingestion")
	// Placeholder for PR 2
	telemetry.Info("ingestion_task_finished")
}
