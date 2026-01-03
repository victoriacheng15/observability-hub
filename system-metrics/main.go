package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"logger"
	"system-metrics/collectors"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/host"
)

func main() {
	// Initialize structured logging
	logger.Setup("system-metrics")

	// Load .env (current or parent)
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	// 1. Initial Detection
	hInfo, err := host.Info()
	if err != nil {
		slog.Error("host_info_failed", "error", err)
		os.Exit(1)
	}
	osName := fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion)

	hostName, _ := os.Hostname()
	if hostName == "" {
		hostName = "homelab"
	}

	// 2. Database Connection
	connStr := getConnStr()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		slog.Error("db_connection_failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// 3. Ensure Schema
	ensureSchema(ctx, conn)

	// 4. Collect and Store Once
	collectAndStore(ctx, conn, hostName, osName)
}

func collectAndStore(ctx context.Context, conn *pgx.Conn, hostName string, osName string) {
	now := time.Now().UTC().Truncate(time.Second)

	// Collect
	cpu, _ := collectors.GetCPUStats()
	mem, _ := collectors.GetMemoryStats()
	disk, _ := collectors.GetDiskStats()
	net, _ := collectors.GetNetworkStats()

	// Store to DB
	metrics := []struct {
		mType   string
		payload interface{}
	}{
		{"cpu", cpu},
		{"memory", mem},
		{"disk", disk},
		{"network", net},
	}

	var insertErrors []string
	for _, m := range metrics {
		if m.payload == nil {
			continue
		}
		payloadJSON, _ := json.Marshal(m.payload)
		_, err := conn.Exec(ctx,
			"INSERT INTO system_metrics (time, host, os, metric_type, payload) VALUES ($1, $2, $3, $4, $5)",
			now, hostName, osName, m.mType, payloadJSON,
		)
		if err != nil {
			slog.Error("db_insert_failed", "metric_type", m.mType, "error", err)
			insertErrors = append(insertErrors, err.Error())
		}
	}

	// Log success only at the top of the hour and if no errors
	if now.Minute() == 0 {
		if len(insertErrors) == 0 {
			slog.Info("metrics_collected", "status", "success")
		} else {
			slog.Warn("metrics_collected", "status", "partial_failure", "error_count", len(insertErrors))
		}
	}
}

func ensureSchema(ctx context.Context, conn *pgx.Conn) {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS system_metrics (
			time TIMESTAMPTZ(0) NOT NULL,
			host TEXT NOT NULL,
			os TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			payload JSONB NOT NULL
		);
	`)
	if err != nil {
		slog.Error("schema_init_failed", "error", err)
		os.Exit(1)
	}

	// Enable hypertable if TimescaleDB is available
	_, err = conn.Exec(ctx, "SELECT create_hypertable('system_metrics', 'time', if_not_exists => true);")
	if err != nil {
		// Just info, as we might be running on standard Postgres
		slog.Info("hypertable_check", "status", "skipped_or_failed", "detail", err)
	}
}

func getConnStr() string {
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		return connStr
	}

	host := getEnv("DB_HOST")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER")
	dbname := getEnv("DB_NAME")
	password := os.Getenv("SERVER_DB_PASSWORD")

	if host == "" {
		slog.Error("env_var_missing", "key", "DB_HOST")
		os.Exit(1)
	}
	if user == "" {
		slog.Error("env_var_missing", "key", "DB_USER")
		os.Exit(1)
	}
	if dbname == "" {
		slog.Error("env_var_missing", "key", "DB_NAME")
		os.Exit(1)
	}
	if password == "" {
		slog.Error("env_var_missing", "key", "SERVER_DB_PASSWORD")
		os.Exit(1)
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

func getEnv(key string, fallback ...string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}
