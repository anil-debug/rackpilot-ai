package fixtures

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSubsetIntegrity(t *testing.T) {
	store, err := NewStore(DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateSubsetIntegrity(); err != nil {
		t.Fatal(err)
	}
}

func TestLicenseAndNoticePresent(t *testing.T) {
	base := filepath.Dir(DefaultDir())
	for _, name := range []string{"LICENSE.BSD", "NOTICE.md"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestContainerFixtureRootWhenPresent(t *testing.T) {
	root := "/fixtures/dmtf/public-rackmount1"
	if _, err := os.Stat(root); err != nil {
		t.Skip("container fixture path not present on host")
	}
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateSubsetIntegrity(); err != nil {
		t.Fatal(err)
	}
}
