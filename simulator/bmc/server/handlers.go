package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/scenario"
)

type scenarioRequest struct {
	ComponentID string `json:"component_id"`
}

type Handler struct {
	eng     *engine.Engine
	scn     *scenario.Manager
	cfg     config.Config
	version string
}

func NewHandler(cfg config.Config, eng *engine.Engine, scn *scenario.Manager, version string) *Handler {
	return &Handler{eng: eng, scn: scn, cfg: cfg, version: version}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"status":          "ok",
		"service":         "rackpilot-bmc-simulator",
		"simulation_mode": string(h.cfg.Mode),
		"redfish":         "/redfish/v1/",
		"version":         h.version,
	}
	if h.cfg.Mode == config.ModeStatic {
		resp["fixture_dataset"] = "dmtf-public-rackmount1"
		resp["synthetic_telemetry"] = false
		resp["live_simulation"] = false
	} else {
		resp["synthetic_telemetry"] = true
		resp["live_simulation"] = true
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) requireDynamic(w http.ResponseWriter) bool {
	if h.cfg.Mode == config.ModeStatic {
		writeErr(w, http.StatusConflict, domain.APIError{
			Code:    "static_mode",
			Message: "fault scenarios require dynamic simulation mode (RACKPILOT_BMC_MODE=dynamic)",
		})
		return false
	}
	return true
}

func (h *Handler) ScenarioFan(w http.ResponseWriter, r *http.Request) {
	if !h.requireDynamic(w) {
		return
	}
	h.apply(w, r, h.scn.FanFailure, "fan_failure")
}

func (h *Handler) ScenarioPSU(w http.ResponseWriter, r *http.Request) {
	if !h.requireDynamic(w) {
		return
	}
	h.apply(w, r, h.scn.PSUFailure, "psu_failure")
}

func (h *Handler) ScenarioCPU(w http.ResponseWriter, r *http.Request) {
	if !h.requireDynamic(w) {
		return
	}
	h.apply(w, r, h.scn.CPUOverheat, "cpu_overheating")
}

func (h *Handler) ScenarioMemory(w http.ResponseWriter, r *http.Request) {
	if !h.requireDynamic(w) {
		return
	}
	h.apply(w, r, h.scn.MemoryError, "memory_error")
}

func (h *Handler) ScenarioReset(w http.ResponseWriter, r *http.Request) {
	if !h.requireDynamic(w) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.scn.Reset()
	writeJSON(w, http.StatusOK, map[string]any{
		"scenario":            "reset",
		"synthetic_telemetry": true,
		"message":             "Synthetic BMC hardware restored",
	})
}

func (h *Handler) apply(w http.ResponseWriter, r *http.Request, fn func(string) error, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req scenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, domain.ErrInvalidJSON)
		return
	}
	if err := fn(req.ComponentID); err != nil {
		status := http.StatusBadRequest
		var api domain.APIError
		if errors.As(err, &api) {
			if api.Code == "not_found" {
				status = http.StatusNotFound
			}
			writeErr(w, status, api)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeErr(w, http.StatusNotFound, domain.ErrNotFound)
			return
		}
		writeErr(w, http.StatusBadRequest, domain.ErrInvalidID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scenario":            name,
		"component_id":        req.ComponentID,
		"synthetic_telemetry": true,
		"message":             "Scenario active",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, e domain.APIError) {
	writeJSON(w, status, map[string]any{"error": e})
}
