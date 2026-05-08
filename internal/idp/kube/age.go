package kube

import (
	"fmt"
	"time"
)

func Age(now time.Time, created time.Time) string {
	if created.IsZero() {
		return "unknown"
	}

	duration := now.Sub(created)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}
