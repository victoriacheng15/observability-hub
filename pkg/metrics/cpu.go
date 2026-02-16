package metrics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

type CPUStats struct {
	Usage       float64            `json:"usage"` // Total usage %
	PackageTemp float64            `json:"package_temp"`
	CoreTemps   map[string]float64 `json:"core_temps"`
}

func GetCPUStats() (*CPUStats, error) {
	stats := &CPUStats{
		CoreTemps: make(map[string]float64),
	}

	// 1. Get CPU Usage (Total)
	// We use 1 second to get a meaningful utilization sample
	totalPercent, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get cpu percent: %w", err)
	}
	if len(totalPercent) > 0 {
		stats.Usage = totalPercent[0]
	}

	// 2. Get Temperatures (Manual Parsing)
	temps := getCPUTemperaturesManual()

	// Map manual results to struct
	for label, temp := range temps {
		if label == "package" {
			stats.PackageTemp = temp
		} else {
			stats.CoreTemps[label] = temp
		}
	}

	return stats, nil
}

// getCPUTemperaturesManual parses /sys/class/hwmon manually.
func getCPUTemperaturesManual() map[string]float64 {
	temps := make(map[string]float64)

	hwmonDir := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonDir)
	if err != nil {
		return temps
	}

	reTempInput := regexp.MustCompile(`^temp(\d+)_input$`)
	reTempLabel := regexp.MustCompile(`^temp(\d+)_label$`)

	for _, entry := range entries {
		hwmonPath := filepath.Join(hwmonDir, entry.Name())
		nameFile := filepath.Join(hwmonPath, "name")
		nameBytes, err := os.ReadFile(nameFile)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameBytes))

		if name != "coretemp" {
			continue
		}

		labels := make(map[string]string)
		inputs := make(map[string]string)

		files, err := os.ReadDir(hwmonPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			filename := file.Name()
			if matches := reTempLabel.FindStringSubmatch(filename); matches != nil {
				index := matches[1]
				labelBytes, _ := os.ReadFile(filepath.Join(hwmonPath, filename))
				label := strings.TrimSpace(string(labelBytes))
				if strings.HasPrefix(label, "Core ") {
					coreNum := strings.TrimPrefix(label, "Core ")
					labels[index] = "core_" + coreNum
				} else if strings.HasPrefix(label, "Package") {
					labels[index] = "package"
				}
			} else if matches := reTempInput.FindStringSubmatch(filename); matches != nil {
				index := matches[1]
				inputBytes, _ := os.ReadFile(filepath.Join(hwmonPath, filename))
				inputs[index] = strings.TrimSpace(string(inputBytes))
			}
		}

		for idx, rawTemp := range inputs {
			tempMilli, err := strconv.ParseFloat(rawTemp, 64)
			if err != nil {
				continue
			}
			tempC := tempMilli / 1000.0
			if label, ok := labels[idx]; ok {
				temps[label] = tempC
			}
		}
	}

	return temps
}
