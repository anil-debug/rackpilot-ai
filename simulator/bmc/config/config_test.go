package config

import "testing"

func TestDefaultModeDynamic(t *testing.T) {
	cfg := Default()
	if cfg.Mode != ModeDynamic {
		t.Fatalf("default mode %q", cfg.Mode)
	}
}
