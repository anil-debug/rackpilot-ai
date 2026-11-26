package store

import (
	"testing"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/correlate"
	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
)

func TestDuplicateSignalUpdatesNotReopens(t *testing.T) {
	mem := NewMemory()
	now := time.Now().UTC()
	r := correlate.Result{
		RuleID: "hardware_cooling", DedupeKey: "hardware_cooling:fan-0", Active: true,
		Incident: inc.Incident{IncidentID: "inc-1", CorrelationRule: "hardware_cooling"},
	}
	o1, _, _ := mem.ApplyTick([]correlate.Result{r}, now, time.Minute)
	o2, u2, _ := mem.ApplyTick([]correlate.Result{r}, now.Add(time.Second), time.Minute)
	if o1 != 1 || o2 != 0 || u2 != 1 {
		t.Fatalf("open1=%d open2=%d updated=%d", o1, o2, u2)
	}
}

func TestRecoveryRequiresFullWindow(t *testing.T) {
	mem := NewMemory()
	now := time.Now().UTC()
	recovery := 2 * time.Minute
	r := correlate.Result{
		RuleID: "hardware_cooling", DedupeKey: "hardware_cooling:fan-0", Active: true,
		Incident: inc.Incident{IncidentID: "inc-1", Status: inc.StatusOpen},
	}
	mem.ApplyTick([]correlate.Result{r}, now, recovery)
	_, _, res1 := mem.ApplyTick(nil, now.Add(30*time.Second), recovery)
	if res1 != 0 {
		t.Fatal("should not resolve before window")
	}
	_, _, res2 := mem.ApplyTick(nil, now.Add(3*time.Minute), recovery)
	if res2 != 1 {
		t.Fatal("should resolve after window")
	}
}
