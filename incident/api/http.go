package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/config"
	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/incident/store"
)

type Handler struct {
	cfg   config.Config
	store *store.Memory
	last  func() (time.Time, string)
}

func NewHandler(cfg config.Config, st *store.Memory, last func() (time.Time, string)) *Handler {
	return &Handler{cfg: cfg, store: st, last: last}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("GET /api/v1/incidents", h.listIncidents)
	mux.HandleFunc("GET /api/v1/incidents/active", h.activeIncidents)
	mux.HandleFunc("GET /api/v1/incidents/{id}", h.getIncident)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	last, errMsg := h.last()
	status := "ok"
	code := http.StatusOK
	if errMsg != "" && time.Since(last) > 2*h.cfg.PollInterval {
		status = "degraded"
		code = http.StatusOK
	}
	writeJSON(w, code, map[string]any{
		"status":     status,
		"service":    "rackpilot-incident-engine",
		"version":    h.cfg.ServiceVersion,
		"last_run":   last.UTC().Format(time.RFC3339),
		"last_error": errMsg,
	})
}

func (h *Handler) listIncidents(w http.ResponseWriter, r *http.Request) {
	offset, limit := pageParams(r)
	items := h.store.List(offset, limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"data":       items,
		"pagination": map[string]int{"offset": offset, "limit": limit, "count": len(items)},
	})
}

func (h *Handler) activeIncidents(w http.ResponseWriter, r *http.Request) {
	items := h.store.Active()
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "count": len(items)})
}

func (h *Handler) getIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing incident id")
		return
	}
	inst, ok := h.store.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "incident not found")
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

func pageParams(r *http.Request) (int, int) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return offset, limit
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

var _ = inc.StatusOpen
