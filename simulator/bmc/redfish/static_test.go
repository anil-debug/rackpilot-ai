package redfish

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/fixtures"
)

func TestStaticRouterDMTFPaths(t *testing.T) {
	store, err := fixtures.NewStore(fixtures.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	r := NewStaticRouter(store)
	mux := http.NewServeMux()
	r.Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	manifest, err := store.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	for path := range manifest {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s -> %d", path, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("%s content-type %q", path, ct)
		}
		if resp.Header.Get("OData-Version") != "4.0" {
			t.Fatalf("%s missing OData-Version", path)
		}
		want, err := store.Read(path)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !bytes.Equal(got, want) {
			t.Fatalf("%s body changed on wire", path)
		}
	}
}

func TestStaticRouterNotFound(t *testing.T) {
	store, err := fixtures.NewStore(fixtures.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	r := NewStaticRouter(store)
	mux := http.NewServeMux()
	r.Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	for _, p := range []string{
		"/redfish/v1/AccountService",
		"/redfish/v1/Systems/missing",
	} {
		resp, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s -> %d want 404", p, resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Fatalf("404 content-type")
		}
	}
}
