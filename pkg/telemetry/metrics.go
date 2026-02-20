package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	metricapi "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

// GetMeter returns a meter with the provided name.
func GetMeter(name string) metricapi.Meter {
	if name == "" {
		name = ScopeName
	}
	return otel.Meter(name)
}

// NewInt64Counter creates an int64 counter with an optional description.
func NewInt64Counter(meter metricapi.Meter, name, description string) (metricapi.Int64Counter, error) {
	opts := []metricapi.Int64CounterOption{}
	if description != "" {
		opts = append(opts, metricapi.WithDescription(description))
	}
	return meter.Int64Counter(name, opts...)
}

// NewInt64Histogram creates an int64 histogram with optional description and unit.
func NewInt64Histogram(meter metricapi.Meter, name, description, unit string) (metricapi.Int64Histogram, error) {
	opts := []metricapi.Int64HistogramOption{}
	if description != "" {
		opts = append(opts, metricapi.WithDescription(description))
	}
	if unit != "" {
		opts = append(opts, metricapi.WithUnit(unit))
	}
	return meter.Int64Histogram(name, opts...)
}

// AddInt64Counter adds a value to an int64 counter with optional attributes.
func AddInt64Counter(ctx context.Context, counter metricapi.Int64Counter, value int64, attrs ...Attribute) {
	counter.Add(ctx, value, metricapi.WithAttributes(attrs...))
}

// RecordInt64Histogram records a value in an int64 histogram with optional attributes.
func RecordInt64Histogram(ctx context.Context, histogram metricapi.Int64Histogram, value int64, attrs ...Attribute) {
	histogram.Record(ctx, value, metricapi.WithAttributes(attrs...))
}

func initMetrics(ctx context.Context, conn *grpc.ClientConn, res *resource.Resource) (func(context.Context) error, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(3*time.Second)),
		),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}
