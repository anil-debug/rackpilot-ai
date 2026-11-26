package analyze

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/investigation/collector"
	"github.com/anil-debug/rackpilot-ai/investigation/config"
	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
	"github.com/anil-debug/rackpilot-ai/investigation/report"
)

func TestLLMRejectsInventedEvidenceID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		payload := map[string]any{
			"choices": []any{
				map[string]any{"message": map[string]string{
					"content": `{"potential_root_causes":["x"],"supporting_evidence_ids":["E999"],"missing_evidence":[],"recommended_next_checks":[],"confidence":"low"}`,
				}},
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.LLMAPIKey = "test"
	cfg.LLMBaseURL = srv.URL
	a := NewLLMAnalyzer(cfg)
	base := report.Investigation{}
	collected := collector.Result{Items: []evidence.Item{{EvidenceID: "E001"}}}
	_, err := a.Enrich(context.Background(), inc.Incident{}, collected, base)
	if err == nil {
		t.Fatal("expected rejection of unknown evidence id")
	}
}

func TestLLMUnavailableFallsBack(t *testing.T) {
	cfg := config.Default()
	cfg.LLMAPIKey = ""
	a := NewLLMAnalyzer(cfg)
	_, err := a.Enrich(context.Background(), inc.Incident{}, collector.Result{}, report.Investigation{})
	if err == nil {
		t.Fatal("expected missing key error")
	}
}
