package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
)

func testHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := config.Default()
	cfg.GPUCount = 4
	eng := engine.New(cfg)
	srv := New(cfg, eng, slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	return httptest.NewServer(srv.httpServer.Handler)
}

func TestHealthEndpoint(t *testing.T) {
	ts := testHTTPServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["synthetic_telemetry"] != true {
		t.Fatal("missing synthetic flag")
	}
}

func TestListAndGetGPU(t *testing.T) {
	ts := testHTTPServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/gpus")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	resp2, err := http.Get(ts.URL + "/api/v1/gpus/gpu-0")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp2.StatusCode)
	}

	resp3, err := http.Get(ts.URL + "/api/v1/gpus/gpu-404")
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp3.StatusCode)
	}
}

func TestOverheatingAPI(t *testing.T) {
	ts := testHTTPServer(t)
	defer ts.Close()

	payload := []byte(`{"gpu_id":"gpu-0"}`)
	resp, err := http.Post(ts.URL+"/api/v1/scenarios/overheating", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestInvalidGPUScenario(t *testing.T) {
	ts := testHTTPServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/v1/scenarios/memory-exhaustion", "application/json", bytes.NewReader([]byte(`{"gpu_id":"gpu-404"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestResetAPI(t *testing.T) {
	ts := testHTTPServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/v1/scenarios/reset", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
