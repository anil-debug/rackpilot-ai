package fixtures

import (
	"fmt"
	"os"
)

// membersCountExceptions documents unmodified DMTF mockup count quirks in the upstream dataset.
var membersCountExceptions = map[string]bool{
	"/redfish/v1/UpdateService/FirmwareInventory": true,
}

// Manifest builds the set of HTTP paths served by the fixture store.
func (s *Store) Manifest() (map[string]string, error) {
	files, err := s.ListJSONFiles()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(files))
	for _, abs := range files {
		p := HTTPPathFromFile(s.root, abs)
		out[p] = abs
	}
	return out, nil
}

// ValidateSubsetIntegrity checks parse, Members, counts, and in-subset links.
func (s *Store) ValidateSubsetIntegrity() error {
	manifest, err := s.Manifest()
	if err != nil {
		return err
	}
	for path, abs := range manifest {
		raw, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		doc, err := ParseObject(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if _, ok := doc["@odata.type"]; !ok {
			return fmt.Errorf("%s: missing @odata.type", path)
		}
		if odataID, ok := doc["@odata.id"].(string); ok {
			want := NormalizeOdataID(odataID)
			if want != path && want+"/" != path {
				return fmt.Errorf("%s: @odata.id %q does not match HTTP path %q", path, odataID, path)
			}
		}
		if members, ok := doc["Members"].([]any); ok {
			if count, ok := doc["Members@odata.count"].(float64); ok {
				if int(count) != len(members) && !membersCountExceptions[path] {
					return fmt.Errorf("%s: Members@odata.count=%v != len(Members)=%d", path, count, len(members))
				}
			}
			for _, id := range MemberOdataIDs(doc) {
				if _, ok := manifest[id]; !ok {
					return fmt.Errorf("%s: member link %q not in fixture subset", path, id)
				}
			}
		}
	}
	rootNav := []string{"/redfish/v1/Systems", "/redfish/v1/Chassis", "/redfish/v1/Managers", "/redfish/v1/UpdateService"}
	body, err := s.Read("/redfish/v1/")
	if err != nil {
		return err
	}
	rootDoc, err := ParseObject(body)
	if err != nil {
		return err
	}
	for _, key := range []string{"Systems", "Chassis", "Managers", "UpdateService"} {
		m, ok := rootDoc[key].(map[string]any)
		if !ok {
			return fmt.Errorf("service root missing %s", key)
		}
		id, ok := m["@odata.id"].(string)
		if !ok {
			return fmt.Errorf("service root %s missing @odata.id", key)
		}
		norm := NormalizeOdataID(id)
		found := false
		for _, want := range rootNav {
			if norm == want {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unexpected root nav id %q", id)
		}
		if _, ok := manifest[norm]; !ok {
			return fmt.Errorf("root link %q not in manifest", norm)
		}
	}
	return nil
}
