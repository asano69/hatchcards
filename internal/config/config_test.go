package config

import "testing"

// TestLoadHooksDirDefault verifies that HooksDir defaults to "./hooks" when
// HOOKS_DIR is unset, matching every installation predating the hooks
// feature (which never sets this variable).
func TestLoadHooksDirDefault(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Data.HooksDir != "./hooks" {
		t.Errorf("HooksDir = %q, want %q", cfg.Data.HooksDir, "./hooks")
	}
}

// TestLoadHooksDirOverride verifies that HOOKS_DIR overrides the default.
func TestLoadHooksDirOverride(t *testing.T) {
	t.Setenv("HOOKS_DIR", "/etc/hatchards/hooks")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Data.HooksDir != "/etc/hatchards/hooks" {
		t.Errorf("HooksDir = %q, want %q", cfg.Data.HooksDir, "/etc/hatchards/hooks")
	}
}
