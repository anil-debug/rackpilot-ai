package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/scenario"
)

const disclaimer = "All telemetry is synthetically simulated; it is not measured from physical GPUs."

type scenarioRequest struct {
	GPUID string `json:"gpu_id"`
}

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	eng     *engine.Engine
	scn     *scenario.Manager
	version string
}

// NewHandler constructs API handlers.
func NewHandler(eng *engine.Engine, scn *scenario.Manager, version string) *Handler {
	return &Handler{eng: eng, scn: scn, version: version}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":              "ok",
		"service":             "rackpilot-gpu-simulator",
		"synthetic_telemetry": true,
		"version":             h.version,
	})
}

func (h *Handler) ListGPUs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"disclaimer": disclaimer,
		"gpus":       h.eng.Snapshot(),
	})
}

func (h *Handler) GetGPU(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, ok := h.eng.Get(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, domain.ErrGPUNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"disclaimer": disclaimer,
		"gpu":        t,
	})
}

func (h *Handler) ScenarioOverheating(w http.ResponseWriter, r *http.Request) {
	h.applyScenario(w, r, h.scn.Overheating, "overheating")
}

func (h *Handler) ScenarioMemoryExhaustion(w http.ResponseWriter, r *http.Request) {
	h.applyScenario(w, r, h.scn.MemoryExhaustion, "memory_exhaustion")
}

func (h *Handler) ScenarioReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	n := h.scn.Reset()
	writeJSON(w, http.StatusOK, map[string]any{
		"scenario":            "reset",
		"message":             "All GPUs restored to baseline simulation",
		"affected_gpu_count":  n,
		"synthetic_telemetry": true,
	})
}

func (h *Handler) applyScenario(w http.ResponseWriter, r *http.Request, apply func(string) error, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req scenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, domain.ErrInvalidJSON)
		return
	}
	if req.GPUID == "" {
		writeAPIError(w, http.StatusBadRequest, domain.ErrInvalidGPUID)
		return
	}
	if err := apply(req.GPUID); err != nil {
		var apiErr domain.APIError
		if errors.As(err, &apiErr) {
			status := http.StatusBadRequest
			if apiErr.Code == "gpu_not_found" {
				status = http.StatusNotFound
			}
			writeAPIError(w, status, apiErr)
			return
		}
		if errors.Is(err, domain.ErrGPUNotFound) {
			writeAPIError(w, http.StatusNotFound, domain.ErrGPUNotFound)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scenario":            name,
		"gpu_id":              req.GPUID,
		"message":             "Scenario active (synthetic telemetry)",
		"synthetic_telemetry": true,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, err domain.APIError) {
	writeJSON(w, status, map[string]any{"error": err})
}
