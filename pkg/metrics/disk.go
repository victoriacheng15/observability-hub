package metrics

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

type PartitionStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskStats map[string]PartitionStats

func GetDiskStats() (DiskStats, error) {
	stats := make(DiskStats)

	// Get all partitions
	partitions, err := disk.Partitions(false) // false means only physical partitions
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	for _, p := range partitions {
		// Filter out boot partitions as they are static and less relevant for monitoring
		if strings.HasPrefix(p.Mountpoint, "/boot") {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			// Skip partitions that can't be read (e.g., permission denied)
			continue
		}

		stats[p.Mountpoint] = PartitionStats{
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		}
	}

	return stats, nil
}
