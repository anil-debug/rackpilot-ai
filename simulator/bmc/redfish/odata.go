package redfish

import (
	"fmt"
	"net/http"
)

const schemaHost = "https://redfish.dmtf.org/schemas/v1"

type Handler struct {
	base string
}

func NewHandler(publicBaseURL string) *Handler {
	base := publicBaseURL
	if base == "" {
		base = "" // relative @odata.id
	}
	return &Handler{base: base}
}

func (h *Handler) path(p string) string {
	if h.base == "" {
		return p
	}
	return h.base + p
}

func statusBlock(health string) map[string]any {
	state := "Enabled"
	if health == "Critical" {
		state = "UnavailableOffline"
	}
	return map[string]any{
		"Health": health,
		"State":  state,
	}
}

func odataServiceRoot() map[string]any {
	return map[string]any{
		"@odata.type": schemaHost + "/ServiceRoot.v1_15_0.ServiceRoot",
	}
}

func notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, `{"error":{"code":"Base.1.0.ResourceNotFound","message":"The requested resource was not found."}}`)
}
