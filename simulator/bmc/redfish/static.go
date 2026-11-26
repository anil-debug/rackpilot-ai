package redfish

import (
	"net/http"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/fixtures"
)

// StaticRouter serves official DMTF mockup JSON without modification.
type StaticRouter struct {
	store *fixtures.Store
}

func NewStaticRouter(store *fixtures.Store) *StaticRouter {
	return &StaticRouter{store: store}
}

func (r *StaticRouter) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /redfish/v1/", r.serve)
	mux.HandleFunc("GET /redfish/v1", func(w http.ResponseWriter, req *http.Request) {
		r.servePath(w, "/redfish/v1/")
	})
}

func (r *StaticRouter) serve(w http.ResponseWriter, req *http.Request) {
	r.servePath(w, req.URL.Path)
}

func (r *StaticRouter) servePath(w http.ResponseWriter, path string) {
	body, err := r.store.Read(path)
	if err != nil {
		notFound(w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("OData-Version", "4.0")
	w.Header().Set("X-RackPilot-Fixture", "dmtf-public-rackmount1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
