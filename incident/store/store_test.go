package store

import (
	"testing"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/correlate"
	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
)

func TestRecoveryAndDedup(t *testing.T) {
	mem := NewMemory()
	now := time.Now().UTC()
	r := correlate.Result{
		RuleID:    "hardware_cooling",
		DedupeKey: "hardware_cooling:fan-0",
		Active:    true,
		Incident: inc.Incident{
			IncidentID: "inc-hardware_cooling-fan-0",
			Title:      "cooling",
			Status:     inc.StatusOpen,
		},
	}
	opened, _, _ := mem.ApplyTick([]correlate.Result{r}, now, time.Minute)
	if opened != 1 {
		t.Fatalf("opened=%d", opened)
	}
	opened2, _, _ := mem.ApplyTick([]correlate.Result{r}, now.Add(5*time.Second), time.Minute)
	if opened2 != 0 {
		t.Fatalf("duplicate open %d", opened2)
	}
	if len(mem.Active()) != 1 {
		t.Fatal("expected one active")
	}
	_, _, _ = mem.ApplyTick(nil, now.Add(30*time.Second), time.Minute)
	_, _, resolved := mem.ApplyTick(nil, now.Add(2*time.Minute), time.Minute)
	if resolved != 1 {
		t.Fatalf("resolved=%d", resolved)
	}
}
