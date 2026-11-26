package fixtures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store maps Redfish URL paths to on-disk mockup JSON (index.json layout).
type Store struct {
	root string
}

// NewStore loads fixtures from root (public-rackmount1 directory).
func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("fixture root required")
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("fixture root not found: %s", root)
	}
	return &Store{root: filepath.Clean(root)}, nil
}

// Root returns the fixture directory.
func (s *Store) Root() string { return s.root }

// Resolve maps /redfish/v1/... to a file path.
func (s *Store) Resolve(redfishURLPath string) (string, error) {
	p := NormalizeOdataID(redfishURLPath)
	p = strings.TrimPrefix(p, "/redfish/v1")
	p = strings.Trim(p, "/")
	rel := "index.json"
	if p != "" {
		rel = filepath.Join(filepath.FromSlash(p), "index.json")
	}
	full := filepath.Join(s.root, rel)
	if _, err := os.Stat(full); err != nil {
		return "", err
	}
	return full, nil
}

// Read returns raw JSON bytes for a Redfish path.
func (s *Store) Read(redfishURLPath string) ([]byte, error) {
	full, err := s.Resolve(redfishURLPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

// ListJSONFiles returns all index.json files under the fixture root.
func (s *Store) ListJSONFiles() ([]string, error) {
	var out []string
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "index.json" {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
