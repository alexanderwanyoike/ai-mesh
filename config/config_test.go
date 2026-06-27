package config

import (
	"os"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	userHomeDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userHomeDir = os.UserHomeDir })

	// Missing file loads empty.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load (empty): %v", err)
	}
	if len(cfg.Keys) != 0 {
		t.Errorf("expected empty keys, got %v", cfg.Keys)
	}

	cfg.Keys["fal"] = "fal-secret"
	cfg.Keys["meshy"] = "meshy-secret"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File is created with 0600 permissions.
	path, _ := Path()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config perms = %o, want 600", perm)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Keys["fal"] != "fal-secret" || loaded.Keys["meshy"] != "meshy-secret" {
		t.Errorf("round-trip mismatch: %v", loaded.Keys)
	}
}

func TestResolveKey(t *testing.T) {
	cases := []struct {
		name           string
		flag, env, cfg string
		want           string
	}{
		{"flag wins", "F", "E", "C", "F"},
		{"env over config", "", "E", "C", "E"},
		{"config fallback", "", "", "C", "C"},
		{"none", "", "", "", ""},
	}
	for _, c := range cases {
		if got := ResolveKey(c.flag, c.env, c.cfg); got != c.want {
			t.Errorf("%s: ResolveKey(%q,%q,%q) = %q, want %q", c.name, c.flag, c.env, c.cfg, got, c.want)
		}
	}
}
