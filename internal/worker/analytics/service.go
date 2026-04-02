package analytics

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"observability-hub/internal/telemetry"
	"observability-hub/internal/worker/store"
)

const (
	ServiceName = "worker.analytics" // Matches updated naming convention
	Interval    = 15 * time.Minute
	Step        = "1m"
)

var (
	pathHostname  = "/etc/host_hostname"
	pathOSRelease = "/etc/host_os-release"
)

// ResourceProvider defines the interface for fetching raw consumption data.
type ResourceProvider interface {
	GetEnergyJoules(ctx context.Context, start, end time.Time) (float64, error)
	GetContainerEnergy(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetHostServiceCPU(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetValueUnits(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetCarbonIntensity(ctx context.Context) (float64, error) // gCO2 per kWh
	GetCostFactor(ctx context.Context) (float64, error)      // CAD per Joule
}

type Service struct {
	Store     *store.Store
	Resources ResourceProvider
}

// Run executes a single one-shot batch of the analytics engine.
func (s *Service) Run(ctx context.Context) error {
	meter := telemetry.GetMeter("worker.analytics")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "worker.analytics.run.duration", "Execution time", "ms")

	start := time.Now()
	tracer := telemetry.GetTracer(ServiceName)
	ctx, span := tracer.Start(ctx, "job.run_batch")
	defer span.End()

	telemetry.Info("analytics_batch_starting")

	// 1. Determine the 15-minute time window
	end := time.Now().UTC().Truncate(Interval)
	startBoundary := end.Add(-Interval)

	// 2. Get Host Metadata
	hostName, osName, err := getHostMetadata()
	if err != nil {
		return fmt.Errorf("failed to detect host metadata: %w", err)
	}

	span.SetAttributes(
		telemetry.StringAttribute("host", hostName),
		telemetry.StringAttribute("os", osName),
		telemetry.StringAttribute("window.start", startBoundary.Format(time.RFC3339)),
		telemetry.StringAttribute("window.end", end.Format(time.RFC3339)),
	)

	// 3. Process Resource Consumption (Energy, Cost, Carbon)
	if s.Resources != nil && s.Store != nil {
		s.processResources(ctx, startBoundary, end, hostName, osName)
	}

	// 4. Process Business Value Units
	if s.Resources != nil && s.Store != nil {
		s.processValueUnits(ctx, startBoundary, end, hostName, osName)
	}

	durationMs := time.Since(start).Milliseconds()
	if durationHist != nil {
		telemetry.RecordInt64Histogram(ctx, durationHist, durationMs)
	}

	telemetry.Info("analytics_batch_complete", "duration_ms", durationMs)
	return nil
}

func (s *Service) processResources(ctx context.Context, start, end time.Time, hostName, osName string) {
	costFactor, _ := s.Resources.GetCostFactor(ctx)
	carbonIntensity, _ := s.Resources.GetCarbonIntensity(ctx)

	// Node Energy
	nodeJoules, err := s.Resources.GetEnergyJoules(ctx, start, end)
	if err == nil && nodeJoules > 0 {
		s.recordMetricsForFeature(ctx, end, "node-total", nodeJoules, costFactor, carbonIntensity, hostName, osName)
	}

	// Pod Energy (Kepler)
	podMetrics, err := s.Resources.GetContainerEnergy(ctx, start, end)
	totalPodJoules := 0.0
	if err == nil {
		for container, joules := range podMetrics {
			featureID := mapContainerToFeature(container)
			if joules > 0 {
				totalPodJoules += joules
				s.recordMetricsForFeature(ctx, end, featureID, joules, costFactor, carbonIntensity, hostName, osName)
			}
		}
	}

	// Host Service Attribution
	hostJoulesGap := nodeJoules - totalPodJoules
	if hostJoulesGap > 0 {
		hostServiceCPU, err := s.Resources.GetHostServiceCPU(ctx, start, end)
		if err == nil {
			totalHostCPU := 0.0
			for _, cpu := range hostServiceCPU {
				totalHostCPU += cpu
			}
			if totalHostCPU > 0 {
				for service, cpu := range hostServiceCPU {
					share := cpu / totalHostCPU
					attributedJoules := hostJoulesGap * share
					featureID := mapContainerToFeature(service)
					s.recordMetricsForFeature(ctx, end, featureID, attributedJoules, costFactor, carbonIntensity, hostName, osName)
				}
			}
		}
	}
}

func (s *Service) processValueUnits(ctx context.Context, start, end time.Time, hostName, osName string) {
	valueUnits, err := s.Resources.GetValueUnits(ctx, start, end)
	if err != nil {
		return
	}

	for feature, count := range valueUnits {
		if count > 0 {
			metadata := map[string]interface{}{"host": hostName, "os": osName}
			_ = s.Store.RecordAnalyticsMetric(ctx, end, feature, store.KindValueUnit, count, "count", metadata)
		}
	}
}

func (s *Service) recordMetricsForFeature(ctx context.Context, t time.Time, featureID string, joules, costFactor, carbonIntensity float64, hostName, osName string) {
	metadata := map[string]interface{}{"host": hostName, "os": osName}
	_ = s.Store.RecordAnalyticsMetric(ctx, t, featureID, store.KindEnergy, joules, "joules", metadata)
	_ = s.Store.RecordAnalyticsMetric(ctx, t, featureID, store.KindCost, joules*costFactor, "cad", metadata)
	_ = s.Store.RecordAnalyticsMetric(ctx, t, featureID, store.KindCarbon, (joules/3600000.0)*carbonIntensity, "g_co2", metadata)
}

func getHostMetadata() (string, string, error) {
	hostnameBytes, err := os.ReadFile(pathHostname)
	if err != nil {
		return "unknown", "unknown", nil // Fallback for testing/non-k3s
	}
	hostname := strings.TrimSpace(string(hostnameBytes))

	osReleaseBytes, err := os.ReadFile(pathOSRelease)
	if err != nil {
		return hostname, "linux-generic", nil
	}

	var name, version string
	lines := strings.Split(string(osReleaseBytes), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			name = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	return hostname, fmt.Sprintf("%s %s", name, version), nil
}

func mapContainerToFeature(container string) string {
	mapping := map[string]string{
		"ingestion": "ingestion", "proxy": "proxy", "analytics": "analytics-engine",
		"mcp-telemetry": "agentic-telemetry", "mcp-pods": "agentic-kubernetes", "mcp-hub": "agentic-hub",
		"postgresql": "database-core", "grafana": "observability-ui", "loki": "observability-logs",
		"tempo": "observability-traces", "openbao": "security-secrets",
	}
	if feature, ok := mapping[container]; ok {
		return feature
	}
	return container
}
