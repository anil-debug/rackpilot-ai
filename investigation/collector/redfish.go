package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
)

// RedfishCollector gathers read-only BMC Redfish resources.
type RedfishCollector struct {
	base   string
	client *http.Client
}

func NewRedfishCollector(base string) *RedfishCollector {
	return &RedfishCollector{base: base, client: &http.Client{Timeout: 15 * time.Second}}
}

func (r *RedfishCollector) Collect(ctx context.Context) ([]evidence.Item, string, error) {
	paths := []struct {
		path        string
		name        string
		description string
	}{
		{"/health", "bmc_health", "BMC simulator health/mode"},
		{"/redfish/v1/Chassis/rackpilot-chassis-001/Thermal", "chassis_thermal", "Chassis thermal Redfish resource"},
		{"/redfish/v1/Chassis/rackpilot-chassis-001/Power", "chassis_power", "Chassis power Redfish resource"},
		{"/redfish/v1/Systems/rackpilot-server-001/LogServices/EventLog/Entries", "event_log", "System event log entries"},
	}
	var items []evidence.Item
	mode := "dynamic"
	var firstErr error
	for _, p := range paths {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base+p.path, nil)
		if err != nil {
			return items, mode, err
		}
		resp, err := r.client.Do(req)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", p.path, err)
			}
			continue
		}
		body := map[string]any{}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: HTTP %d", p.path, resp.StatusCode)
			}
			continue
		}
		if p.path == "/health" {
			if m, ok := body["simulation_mode"].(string); ok && m == "static" {
				mode = "static"
			}
			items = append(items, evidence.Item{
				Source: "redfish", Timestamp: time.Now().UTC(), Resource: "bmc",
				Name: p.name, Value: fmt.Sprint(body["simulation_mode"]),
				Description: p.description, Provenance: redfishProvenance(mode),
				Reference: p.path,
			})
			continue
		}
		summary := summarizeRedfish(body)
		items = append(items, evidence.Item{
			Source: "redfish", Timestamp: time.Now().UTC(), Resource: "rackpilot-chassis-001",
			Name: p.name, Value: summary, Description: p.description,
			Provenance: redfishProvenance(mode), Reference: p.path,
		})
	}
	return items, mode, firstErr
}

func redfishProvenance(mode string) evidence.Provenance {
	if mode == "static" {
		return evidence.ProvenanceDMTFFixture
	}
	return evidence.ProvenanceRedfishLive
}

func summarizeRedfish(body map[string]any) string {
	if fans, ok := body["Fans"].([]any); ok {
		return fmt.Sprintf("fan_count=%d", len(fans))
	}
	if psus, ok := body["PowerSupplies"].([]any); ok {
		return fmt.Sprintf("psu_count=%d", len(psus))
	}
	if members, ok := body["Members"].([]any); ok {
		return fmt.Sprintf("members=%d", len(members))
	}
	return "redfish_payload"
}
