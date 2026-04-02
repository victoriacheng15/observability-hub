package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/telemetry"
	"observability-hub/internal/worker/store"
)

// MockResources satisfies the ResourceProvider interface.
type MockResources struct {
	GetEnergyJoulesFn    func(ctx context.Context, start, end time.Time) (float64, error)
	GetContainerEnergyFn func(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetHostServiceCPUFn  func(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetValueUnitsFn      func(ctx context.Context, start, end time.Time) (map[string]float64, error)
}

func (m *MockResources) GetEnergyJoules(ctx context.Context, start, end time.Time) (float64, error) {
	if m.GetEnergyJoulesFn != nil {
		return m.GetEnergyJoulesFn(ctx, start, end)
	}
	return 100.0, nil
}

func (m *MockResources) GetContainerEnergy(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	if m.GetContainerEnergyFn != nil {
		return m.GetContainerEnergyFn(ctx, start, end)
	}
	return map[string]float64{"postgresql": 20.0}, nil
}

func (m *MockResources) GetHostServiceCPU(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	if m.GetHostServiceCPUFn != nil {
		return m.GetHostServiceCPUFn(ctx, start, end)
	}
	return map[string]float64{"ingestion.service": 0.5}, nil
}

func (m *MockResources) GetValueUnits(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	if m.GetValueUnitsFn != nil {
		return m.GetValueUnitsFn(ctx, start, end)
	}
	return map[string]float64{"ingestion": 10.0}, nil
}

func (m *MockResources) GetCarbonIntensity(ctx context.Context) (float64, error) { return 150.0, nil }
func (m *MockResources) GetCostFactor(ctx context.Context) (float64, error) {
	return 0.15 / 3600000.0, nil
}

func setupHostMetadata(t *testing.T) func() {
	tmpDir, _ := os.MkdirTemp("", "etc-host")
	hPath := filepath.Join(tmpDir, "host_hostname")
	oPath := filepath.Join(tmpDir, "host_os-release")
	_ = os.WriteFile(hPath, []byte("test-host"), 0644)
	_ = os.WriteFile(oPath, []byte("ID=linux\nVERSION_ID=6.0\n"), 0644)
	oldH, oldO := pathHostname, pathOSRelease
	pathHostname, pathOSRelease = hPath, oPath
	return func() {
		os.RemoveAll(tmpDir)
		pathHostname, pathOSRelease = oldH, oldO
	}
}

func TestService_Run(t *testing.T) {
	cleanup := setupHostMetadata(t)
	defer cleanup()
	telemetry.SilenceLogs()

	mdb, mCleanup := postgres.NewMockDB(t)
	defer mCleanup()
	workerStore := store.NewStore(mdb.Wrapper())

	tests := []struct {
		name      string
		resources ResourceProvider
		setup     func()
		wantErr   bool
	}{
		{
			name:      "Success",
			resources: &MockResources{},
			setup: func() {
				mdb.Mock.ExpectExec("INSERT INTO analytics_metrics").WillReturnResult(mdb.NewResult(1, 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			s := &Service{
				Store:     workerStore,
				Resources: tt.resources,
			}
			err := s.Run(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_ProcessResources(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()
	workerStore := store.NewStore(mdb.Wrapper())

	s := &Service{
		Store:     workerStore,
		Resources: &MockResources{},
	}

	mdb.Mock.ExpectExec("INSERT INTO analytics_metrics").WillReturnResult(mdb.NewResult(1, 1))

	s.processResources(context.Background(), time.Now().Add(-Interval), time.Now(), "host", "os")
}

func TestService_ProcessValueUnits(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()
	workerStore := store.NewStore(mdb.Wrapper())

	s := &Service{
		Store:     workerStore,
		Resources: &MockResources{},
	}

	mdb.Mock.ExpectExec("INSERT INTO analytics_metrics").
		WithArgs(mdb.AnyArg(), "ingestion", "value_unit", 10.0, "count", mdb.AnyArg()).
		WillReturnResult(mdb.NewResult(1, 1))

	s.processValueUnits(context.Background(), time.Now().Add(-Interval), time.Now(), "host", "os")
}
