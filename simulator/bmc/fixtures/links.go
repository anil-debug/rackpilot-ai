package fixtures

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// NormalizeOdataID strips JSON fragments and normalizes /redfish/v1 paths.
func NormalizeOdataID(id string) string {
	base := strings.SplitN(id, "#", 2)[0]
	base = strings.TrimSuffix(base, "/")
	if base == "/redfish/v1" {
		return "/redfish/v1/"
	}
	if base != "" && !strings.HasSuffix(base, "/") {
		return base
	}
	return base
}

// HTTPPathFromFile maps a fixture index.json path to its Redfish URL.
func HTTPPathFromFile(root, absFile string) string {
	rel, err := filepath.Rel(root, absFile)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, "/index.json")
	if rel == "index.json" || rel == "" {
		return "/redfish/v1/"
	}
	return "/redfish/v1/" + rel
}

// ParseObject reads JSON object from bytes.
func ParseObject(b []byte) (map[string]any, error) {
	var doc map[string]any
	err := json.Unmarshal(b, &doc)
	return doc, err
}

// MemberOdataIDs returns @odata.id values from a Members collection.
func MemberOdataIDs(doc map[string]any) []string {
	raw, ok := doc["Members"].([]any)
	if !ok {
		return nil
	}
	var ids []string
	for _, m := range raw {
		ref, ok := m.(map[string]any)
		if !ok {
			continue
		}
		id, ok := ref["@odata.id"].(string)
		if !ok {
			continue
		}
		ids = append(ids, NormalizeOdataID(id))
	}
	return ids
}

// NavLinkIDs returns @odata.id from immediate navigation objects (map values).
func NavLinkIDs(doc map[string]any) []string {
	var ids []string
	for _, v := range doc {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := m["@odata.id"].(string); ok {
			ids = append(ids, NormalizeOdataID(id))
		}
	}
	return ids
}
