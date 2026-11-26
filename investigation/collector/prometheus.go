package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
)

// PromClient queries Prometheus HTTP API (instant + range).
type PromClient struct {
	base   string
	client *http.Client
}

func NewPromClient(base string) *PromClient {
	return &PromClient{base: base, client: &http.Client{Timeout: 60 * time.Second}}
}

type matrixPoint struct {
	Metric map[string]string
	Values [][]any
}

func (p *PromClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]evidence.Item, error) {
	u, err := url.Parse(p.base + "/api/v1/query_range")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("start", fmt.Sprintf("%f", float64(start.Unix())))
	q.Set("end", fmt.Sprintf("%f", float64(end.Unix())))
	q.Set("step", step.String())
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string        `json:"resultType"`
			Result     []matrixPoint `json:"result"`
			Error      string        `json:"error"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status != "success" {
		msg := body.Error
		if msg == "" {
			msg = body.Data.Error
		}
		return nil, fmt.Errorf("prometheus range query failed: %s", msg)
	}

	var items []evidence.Item
	idx := 0
	for _, series := range body.Data.Result {
		name := series.Metric["__name__"]
		if name == "" {
			name = query
		}
		res := labelResource(series.Metric)
		for _, pair := range series.Values {
			if len(pair) < 2 {
				continue
			}
			ts, _ := pair[0].(float64)
			valStr, _ := pair[1].(string)
			idx++
			items = append(items, evidence.Item{
				EvidenceID:  fmt.Sprintf("E%03d", idx),
				Source:      "prometheus",
				Timestamp:   time.Unix(int64(ts), 0).UTC(),
				Resource:    res,
				Name:        name,
				Value:       valStr,
				Description: fmt.Sprintf("Prometheus sample %s=%s", name, valStr),
				Provenance:  promProvenance(series.Metric),
				Reference:   "promql:" + query,
				Labels:      series.Metric,
			})
		}
	}
	return items, nil
}

func (p *PromClient) QueryInstant(ctx context.Context, query string) ([]evidence.Item, error) {
	u, err := url.Parse(p.base + "/api/v1/query")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
			Error string `json:"error"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status != "success" {
		msg := body.Error
		if msg == "" {
			msg = body.Data.Error
		}
		return nil, fmt.Errorf("prometheus instant query failed: %s", msg)
	}
	var items []evidence.Item
	for i, r := range body.Data.Result {
		name := r.Metric["__name__"]
		val := fmt.Sprint(r.Value[1])
		ts, _ := r.Value[0].(float64)
		items = append(items, evidence.Item{
			EvidenceID:  fmt.Sprintf("E%03d", i+1),
			Source:      "prometheus",
			Timestamp:   time.Unix(int64(ts), 0).UTC(),
			Resource:    labelResource(r.Metric),
			Name:        name,
			Value:       val,
			Description: fmt.Sprintf("Instant %s=%s", name, val),
			Provenance:  promProvenance(r.Metric),
			Reference:   "promql:" + query,
			Labels:      r.Metric,
		})
	}
	return items, nil
}

func promProvenance(m map[string]string) evidence.Provenance {
	if m["rackpilot_telemetry"] == "kubernetes" || m["namespace"] != "" && m["cluster"] != "" {
		return evidence.ProvenanceK8sMetrics
	}
	return evidence.ProvenanceSyntheticLive
}

func labelResource(m map[string]string) string {
	for _, k := range []string{"fan_id", "gpu_id", "cpu_id", "deployment", "pod", "container", "server_id"} {
		if v := m[k]; v != "" {
			if m["namespace"] != "" {
				return m["namespace"] + "/" + v
			}
			return v
		}
	}
	if j := m["job"]; j != "" {
		return j
	}
	return "unknown"
}

// ParseFloat helper for comparisons.
func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
