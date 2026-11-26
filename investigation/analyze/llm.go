package analyze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/investigation/collector"
	"github.com/anil-debug/rackpilot-ai/investigation/config"
	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
	"github.com/anil-debug/rackpilot-ai/investigation/report"
)

// LLMAnalyzer optionally enriches deterministic reports via OpenAI-compatible APIs.
type LLMAnalyzer struct {
	cfg    config.Config
	client *http.Client
}

func NewLLMAnalyzer(cfg config.Config) *LLMAnalyzer {
	return &LLMAnalyzer{cfg: cfg, client: &http.Client{Timeout: cfg.LLMTimeout}}
}

type llmPayload struct {
	PotentialRootCauses   []string `json:"potential_root_causes"`
	SupportingEvidenceIDs []string `json:"supporting_evidence_ids"`
	MissingEvidence       []string `json:"missing_evidence"`
	RecommendedChecks     []string `json:"recommended_next_checks"`
	Confidence            string   `json:"confidence"`
}

func (l *LLMAnalyzer) Enrich(ctx context.Context, incident inc.Incident, collected collector.Result, base report.Investigation) (report.Investigation, error) {
	if l.cfg.LLMAPIKey == "" {
		return base, fmt.Errorf("missing LLM API key")
	}
	evidenceJSON, _ := json.Marshal(trimEvidence(collected.Items, 80))
	system := "You are an infrastructure SRE. Output ONLY valid JSON matching the schema. Never invent metrics. Every hypothesis must reference evidence_id values from the input."
	user := fmt.Sprintf(`Incident rule=%s title=%s facts=%v evidence=%s schema keys: potential_root_causes,supporting_evidence_ids,missing_evidence,recommended_next_checks,confidence. Do not list confirmed root causes.`,
		incident.CorrelationRule, incident.Title, incident.ObservedFacts, string(evidenceJSON))

	body := map[string]any{
		"model": l.cfg.LLMModel,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.1,
	}
	raw, _ := json.Marshal(body)
	url := strings.TrimSuffix(l.cfg.LLMBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return base, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.cfg.LLMAPIKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return base, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return base, fmt.Errorf("LLM HTTP %d", resp.StatusCode)
	}

	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chat); err != nil || len(chat.Choices) == 0 {
		return base, fmt.Errorf("invalid LLM response")
	}
	content := strings.TrimSpace(chat.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	var parsed llmPayload
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return base, fmt.Errorf("LLM JSON parse: %w", err)
	}
	valid := evidenceIDSet(collected.Items)
	for _, id := range parsed.SupportingEvidenceIDs {
		if !valid[id] {
			return base, fmt.Errorf("LLM cited unknown evidence id %s", id)
		}
	}

	out := base
	if len(parsed.PotentialRootCauses) > 0 {
		out.PotentialRootCauses = parsed.PotentialRootCauses
	}
	if len(parsed.SupportingEvidenceIDs) > 0 {
		out.SupportingEvidenceIDs = uniqueStrings(parsed.SupportingEvidenceIDs)
	}
	if len(parsed.MissingEvidence) > 0 {
		out.MissingEvidence = uniqueStrings(append(out.MissingEvidence, parsed.MissingEvidence...))
	}
	if len(parsed.RecommendedChecks) > 0 {
		out.RecommendedChecks = uniqueStrings(parsed.RecommendedChecks)
	}
	if parsed.Confidence != "" {
		out.Confidence = parsed.Confidence
	}
	return out, nil
}

func trimEvidence(items []evidence.Item, max int) []evidence.Item {
	if len(items) <= max {
		return items
	}
	return items[len(items)-max:]
}

func evidenceIDSet(items []evidence.Item) map[string]bool {
	m := map[string]bool{}
	for _, it := range items {
		m[it.EvidenceID] = true
	}
	return m
}
