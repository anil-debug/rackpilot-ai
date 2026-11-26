package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
)

// IncidentClient fetches incidents from Milestone 5 engine (read-only).
type IncidentClient struct {
	base   string
	client *http.Client
}

func NewIncidentClient(base string) *IncidentClient {
	return &IncidentClient{base: base, client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *IncidentClient) Get(ctx context.Context, id string) (inc.Incident, error) {
	url := fmt.Sprintf("%s/api/v1/incidents/%s", c.base, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return inc.Incident{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return inc.Incident{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return inc.Incident{}, fmt.Errorf("incident not found")
	}
	if resp.StatusCode >= 400 {
		return inc.Incident{}, fmt.Errorf("incident engine HTTP %d", resp.StatusCode)
	}
	var inst inc.Incident
	if err := json.NewDecoder(resp.Body).Decode(&inst); err != nil {
		return inc.Incident{}, err
	}
	return inst, nil
}
