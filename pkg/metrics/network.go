package metrics

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/net"
)

type InterfaceStats struct {
	BytesRecv uint64 `json:"rx_bytes"`
	BytesSent uint64 `json:"tx_bytes"`
}

type NetworkStats map[string]InterfaceStats

func GetNetworkStats() (NetworkStats, error) {
	stats := make(NetworkStats)

	// Get I/O counters for all interfaces
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("failed to get network io counters: %w", err)
	}

	for _, nic := range ioCounters {
		// Filter out virtual/internal interfaces
		if shouldIgnoreInterface(nic.Name) {
			continue
		}

		stats[nic.Name] = InterfaceStats{
			BytesRecv: nic.BytesRecv,
			BytesSent: nic.BytesSent,
		}
	}

	return stats, nil
}

func shouldIgnoreInterface(name string) bool {
	// Common virtual/loopback/VPN prefixes to ignore
	prefixes := []string{
		"lo",        // Loopback
		"docker",    // Docker bridge
		"veth",      // Virtual ethernet (containers)
		"br-",       // Docker custom bridges
		"tun",       // Tunnels / VPNs
		"tailscale", // Tailscale VPN
		"wg",        // Wireguard
		"zt",        // ZeroTier
		"dummy",     // Dummy interfaces
		"virbr",     // Libvirt bridge
	}

	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
