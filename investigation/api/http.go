package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/anil-debug/rackpilot-ai/investigation/config"
	"github.com/anil-debug/rackpilot-ai/investigation/report"
	"github.com/anil-debug/rackpilot-ai/investigation/service"
)

type Handler struct {
	cfg config.Config
	svc *service.Investigator
}

func NewHandler(cfg config.Config, svc *service.Investigator) *Handler {
	return &Handler{cfg: cfg, svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("POST /api/v1/investigations", h.create)
	mux.HandleFunc("GET /api/v1/investigations", h.list)
	mux.HandleFunc("GET /api/v1/investigations/{id}", h.get)
	mux.HandleFunc("GET /api/v1/investigations/{id}/evidence", h.evidence)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "rackpilot-investigator",
		"version": h.cfg.ServiceVersion,
		"mode":    h.cfg.InvestigatorMode,
	})
}

type createRequest struct {
	IncidentID string `json:"incident_id"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IncidentID == "" {
		writeErr(w, http.StatusBadRequest, "incident_id required")
		return
	}
	inv, err := h.svc.Investigate(r.Context(), req.IncidentID)
	if err != nil {
		if err.Error() == "incident not found" {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	code := http.StatusCreated
	switch inv.Status {
	case report.StatusPartial:
		code = http.StatusPartialContent
	case report.StatusFailed:
		code = http.StatusUnprocessableEntity
	}
	writeJSON(w, code, inv)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	offset, limit := pageParams(r)
	items := h.svc.List(offset, limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"data":       items,
		"pagination": map[string]int{"offset": offset, "limit": limit, "count": len(items)},
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, ok := h.svc.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "investigation not found")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (h *Handler) evidence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, ok := h.svc.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "investigation not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"investigation_id": id,
		"count":            len(inv.EvidenceTimeline),
		"evidence":         inv.EvidenceTimeline,
		"retrieved_at":     time.Now().UTC().Format(time.RFC3339),
	})
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
