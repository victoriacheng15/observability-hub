package kube

import (
	"testing"
	"time"
)

func TestAge(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		created time.Time
		want    string
	}{
		{
			name: "zero timestamp",
			want: "unknown",
		},
		{
			name:    "seconds below one minute",
			created: now.Add(-30 * time.Second),
			want:    "30s",
		},
		{
			name:    "minutes below one hour",
			created: now.Add(-45 * time.Minute),
			want:    "45m",
		},
		{
			name:    "hours below one day",
			created: now.Add(-12 * time.Hour),
			want:    "12h",
		},
		{
			name:    "days after one day",
			created: now.Add(-72 * time.Hour),
			want:    "3d",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Age(now, test.created)
			if got != test.want {
				t.Fatalf("Age() = %q, want %q", got, test.want)
			}
		})
	}
}
