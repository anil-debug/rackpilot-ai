package fixtures

import (
	"os"
	"path/filepath"
	"runtime"
)

const defaultRel = "testdata/redfish/dmtf/public-rackmount1"

// DefaultDir resolves the repository fixture path from this package location.
func DefaultDir() string {
	if v := os.Getenv("RACKPILOT_BMC_FIXTURE_ROOT"); v != "" {
		return v
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return defaultRel
	}
	// simulator/bmc/fixtures -> repo root
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	return filepath.Join(repo, defaultRel)
}
