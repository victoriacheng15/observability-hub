package collectors

import (
	"testing"
)

func TestCollectors(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "GetCPUStats",
			testFunc: func(t *testing.T) {
				stats, err := GetCPUStats()
				if err != nil {
					t.Fatalf("Failed to get CPU stats: %v", err)
				}
				if stats == nil {
					t.Fatal("Expected CPU stats, got nil")
				}
				if stats.Usage < 0 || stats.Usage > 100 {
					t.Errorf("CPU Usage out of bounds (0-100): %f", stats.Usage)
				}
				t.Logf("CPU Usage: %.2f%%", stats.Usage)
			},
		},
		{
			name: "GetMemoryStats",
			testFunc: func(t *testing.T) {
				stats, err := GetMemoryStats()
				if err != nil {
					t.Fatalf("Failed to get Memory stats: %v", err)
				}
				if stats == nil {
					t.Fatal("Expected Memory stats, got nil")
				}
				if stats.Total == 0 {
					t.Error("Total memory reported as 0")
				}
				if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
					t.Errorf("Memory Used Percent out of bounds: %f", stats.UsedPercent)
				}
			},
		},
		{
			name: "GetDiskStats",
			testFunc: func(t *testing.T) {
				stats, err := GetDiskStats()
				if err != nil {
					t.Fatalf("Failed to get Disk stats: %v", err)
				}
				if stats == nil {
					t.Fatal("Expected Disk stats, got nil")
				}
				if len(stats) == 0 {
					t.Log("Warning: No disk partitions found (might be expected in some container envs)")
				}
				for mount, part := range stats {
					if part.Total == 0 {
						t.Errorf("Partition %s has 0 total size", mount)
					}
					if part.UsedPercent < 0 || part.UsedPercent > 100 {
						t.Errorf("Partition %s used percent out of bounds: %f", mount, part.UsedPercent)
					}
				}
			},
		},
		{
			name: "GetNetworkStats",
			testFunc: func(t *testing.T) {
				stats, err := GetNetworkStats()
				if err != nil {
					t.Fatalf("Failed to get Network stats: %v", err)
				}
				if stats == nil {
					t.Fatal("Expected Network stats, got nil")
				}
				for name := range stats {
					if shouldIgnoreInterface(name) {
						t.Errorf("Found ignored interface in output: %s", name)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func TestShouldIgnoreInterface(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"lo", true},
		{"docker0", true},
		{"veth1234", true},
		{"eth0", false},
		{"wlan0", false},
		{"enp3s0", false},
		{"tun0", true},
		{"br-custom", true},
		{"tailscale0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := shouldIgnoreInterface(tt.name); result != tt.expected {
				t.Errorf("shouldIgnoreInterface(%s) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}
