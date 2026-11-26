package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Sample is one instant-query sample.
type Sample struct {
	Metric map[string]string
	Value  float64
	Time   time.Time
}

// Client queries Prometheus HTTP API.
type Client interface {
	QueryInstant(ctx context.Context, promQL string) ([]Sample, error)
}

type httpClient struct {
	base   string
	client *http.Client
}

// NewHTTPClient creates a Prometheus API client.
func NewHTTPClient(baseURL string) Client {
	return &httpClient{
		base:   baseURL,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type queryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
		Error string `json:"error"`
	} `json:"data"`
	Error     string `json:"error"`
	ErrorType string `json:"errorType"`
}

func (c *httpClient) QueryInstant(ctx context.Context, promQL string) ([]Sample, error) {
	u, err := url.Parse(c.base + "/api/v1/query")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", promQL)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body queryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status != "success" {
		msg := body.Error
		if msg == "" {
			msg = body.Data.Error
		}
		return nil, fmt.Errorf("prometheus query failed: %s", msg)
	}
	out := make([]Sample, 0, len(body.Data.Result))
	for _, r := range body.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		ts, ok := r.Value[0].(float64)
		if !ok {
			continue
		}
		var val float64
		switch v := r.Value[1].(type) {
		case string:
			_, _ = fmt.Sscanf(v, "%f", &val)
		case float64:
			val = v
		}
		out = append(out, Sample{
			Metric: r.Metric,
			Value:  val,
			Time:   time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9)),
		})
	}
	return out, nil
}
