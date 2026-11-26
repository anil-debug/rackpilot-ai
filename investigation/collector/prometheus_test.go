package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPromClientQueryRange(t *testing.T) {
	now := float64(time.Now().Unix())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []any{
					map[string]any{
						"metric": map[string]string{"__name__": "rackpilot_bmc_fan_speed_rpm", "fan_id": "fan-0"},
						"values": [][]any{{now, "0"}},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewPromClient(srv.URL)
	start := time.Now().Add(-10 * time.Minute)
	end := time.Now()
	items, err := c.QueryRange(context.Background(), `rackpilot_bmc_fan_speed_rpm`, start, end, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Value != "0" {
		t.Fatalf("unexpected items %+v", items)
	}
}

func TestPromClientQueryError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "error": "bad query"})
	}))
	defer srv.Close()
	_, err := NewPromClient(srv.URL).QueryRange(context.Background(), "bad", time.Now(), time.Now(), time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}
