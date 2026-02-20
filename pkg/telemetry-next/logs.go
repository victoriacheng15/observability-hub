package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	otellog "go.opentelemetry.io/otel/log"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

// GetLogger returns an OpenTelemetry logger with the provided name.
func GetLogger(name string) otellog.Logger {
	if name == "" {
		name = ScopeName
	}
	return otellogglobal.GetLoggerProvider().Logger(name)
}

// Info logs an info-level message using the bridged OTel-slog handler.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs a warn-level message using the bridged OTel-slog handler.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs an error-level message using the bridged OTel-slog handler.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func initLogs(ctx context.Context, conn *grpc.ClientConn, res *resource.Resource) (func(context.Context) error, error) {
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	consoleExporter, err := stdoutlog.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(consoleExporter)),
		sdklog.WithResource(res),
	)
	otellogglobal.SetLoggerProvider(lp)

	slog.SetDefault(slog.New(otelslog.NewHandler(ScopeName, otelslog.WithLoggerProvider(lp))))

	return lp.Shutdown, nil
}
