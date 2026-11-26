package domain

import "testing"

func TestDeriveHealthTemperature(t *testing.T) {
	th := DefaultHealthThresholds()
	tests := []struct {
		temp float64
		want HealthStatus
	}{
		{70, HealthHealthy},
		{85, HealthWarning},
		{95, HealthCritical},
	}
	for _, tc := range tests {
		got := DeriveHealth(tc.temp, 0, 1, th)
		if got != tc.want {
			t.Fatalf("temp %v: got %v want %v", tc.temp, got, tc.want)
		}
	}
}

func TestDeriveHealthMemoryDominates(t *testing.T) {
	th := DefaultHealthThresholds()
	total := uint64(100)
	got := DeriveHealth(50, 96, total, th)
	if got != HealthCritical {
		t.Fatalf("got %v", got)
	}
}

func TestHealthToCode(t *testing.T) {
	if HealthToCode(HealthWarning) != HealthCodeWarning {
		t.Fatal()
	}
}
