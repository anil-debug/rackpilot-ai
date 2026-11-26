package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := config.Default()
	eng := engine.New(cfg)
	srv, err := New(cfg, eng, slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(srv.HTTPHandler())
}

func TestHealthAndScenarios(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}
	resp2, err := http.Post(ts.URL+"/api/v1/scenarios/fan-failure", "application/json", bytes.NewReader([]byte(`{"component_id":"fan-0"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatal(resp2.StatusCode)
	}
}
