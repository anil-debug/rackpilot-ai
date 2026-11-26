package redfish

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
)

func TestRedfishPaths(t *testing.T) {
	eng := engine.New(config.Default())
	r := NewRouter("", eng.Snapshot)
	mux := http.NewServeMux()
	r.Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	paths := []string{
		"/redfish/v1/",
		"/redfish/v1/Systems",
		"/redfish/v1/Systems/rackpilot-server-001",
		"/redfish/v1/Chassis/rackpilot-chassis-001",
		"/redfish/v1/Chassis/rackpilot-chassis-001/Thermal",
		"/redfish/v1/Chassis/rackpilot-chassis-001/Power",
		"/redfish/v1/Managers/BMC",
		"/redfish/v1/UpdateService/FirmwareInventory",
		"/redfish/v1/Systems/rackpilot-server-001/LogServices/EventLog/Entries",
	}
	for _, p := range paths {
		resp, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status %d", p, resp.StatusCode)
		}
	}

	resp, _ := http.Get(ts.URL + "/redfish/v1/Systems/unknown")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", resp.StatusCode)
	}
}
