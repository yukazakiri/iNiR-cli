package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverExternalTargetsFromConfigTargetsDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("INIR_THEME_TARGETS_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(configDir, "xdg-config"))
	configPath := filepath.Join(configDir, "config.json")
	targetsDir := filepath.Join(configDir, "targets")

	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(targetsDir, 0755); err != nil {
		t.Fatalf("create targets dir: %v", err)
	}

	specJSON := `{
	  "id": "custom-demo",
	  "type": "command",
	  "description": "Demo external target",
	  "command": "/usr/bin/true",
	  "args": ["--version"]
	}`

	if err := os.WriteFile(filepath.Join(targetsDir, "custom-demo.json"), []byte(specJSON), 0644); err != nil {
		t.Fatalf("write target spec: %v", err)
	}

	targets, err := discoverExternalTargets(configPath)
	if err != nil {
		t.Fatalf("discover external targets: %v", err)
	}

	spec, ok := targets["custom-demo"]
	if !ok {
		t.Fatalf("expected discovered target custom-demo")
	}
	if spec.Command != "/usr/bin/true" {
		t.Fatalf("unexpected command %q", spec.Command)
	}
}

func TestDiscoverExternalTargetsRejectsDuplicateIDs(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("INIR_THEME_TARGETS_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(configDir, "xdg-config"))
	configPath := filepath.Join(configDir, "config.json")
	targetsDir := filepath.Join(configDir, "targets")

	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(targetsDir, 0755); err != nil {
		t.Fatalf("create targets dir: %v", err)
	}

	first := `{"id":"duplicate-id","type":"command","command":"/bin/true"}`
	second := `{"id":"duplicate-id","type":"command","command":"/usr/bin/true"}`
	if err := os.WriteFile(filepath.Join(targetsDir, "a.json"), []byte(first), 0644); err != nil {
		t.Fatalf("write first spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetsDir, "b.json"), []byte(second), 0644); err != nil {
		t.Fatalf("write second spec: %v", err)
	}

	if _, err := discoverExternalTargets(configPath); err == nil {
		t.Fatalf("expected duplicate id discovery error")
	}
}
