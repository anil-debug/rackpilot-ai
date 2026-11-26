package ingest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/collector/prometheus"
	"github.com/anil-debug/rackpilot-ai/incident/config"
	"github.com/anil-debug/rackpilot-ai/incident/signal"
)

type staticClient map[string][]prometheus.Sample

func (s staticClient) QueryInstant(_ context.Context, promQL string) ([]prometheus.Sample, error) {
	out, ok := s[promQL]
	if !ok {
		return nil, nil
	}
	return out, nil
}

func TestCollectEmptyVectorsNoSignals(t *testing.T) {
	c := New(staticClient{}, config.Default())
	signals, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 0 {
		t.Fatalf("expected no signals, got %d", len(signals))
	}
}

func TestCollectQueryErrorFailsCollect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "error": "timeout"})
	}))
	t.Cleanup(srv.Close)
	c := New(prometheus.NewHTTPClient(srv.URL), config.Default())
	_, err := c.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRestartIncreaseQueryProducesSignal(t *testing.T) {
	now := time.Now()
	samples := map[string][]prometheus.Sample{
		QK8sRestarts: {{
			Metric: map[string]string{"namespace": "rackpilot", "pod": "p1", "container": "c1"},
			Value:  2,
			Time:   now,
		}},
	}
	c := New(staticClient(samples), config.Default())
	signals, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range signals {
		if s.SignalType == signal.TypeRateIncrease {
			found = true
		}
	}
	if !found {
		t.Fatal("expected restart increase signal")
	}
}

func TestBMCMemoryIncreaseQuery(t *testing.T) {
	now := time.Now()
	samples := map[string][]prometheus.Sample{
		QBMCMemErrors: {{
			Metric: map[string]string{"dimm_id": "d0", "server_id": "srv"},
			Value:  3,
			Time:   now,
		}},
	}
	c := New(staticClient(samples), config.Default())
	signals, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("got %d signals", len(signals))
	}
}
