// File: shell_commands_test.go
//
// Tests for shell commands (terminal, close-window, browser).
package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBrowserFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write config with apps.browser
	os.WriteFile(configPath, []byte(`{"apps":{"browser":"firefox"}}`), 0644)
	if got := resolveBrowserFromConfig(configPath); got != "firefox" {
		t.Fatalf("expected firefox, got %q", got)
	}

	// Missing apps.browser
	os.WriteFile(configPath, []byte(`{"apps":{}}`), 0644)
	if got := resolveBrowserFromConfig(configPath); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}

	// Missing file
	if got := resolveBrowserFromConfig(filepath.Join(tmpDir, "missing.json")); got != "" {
		t.Fatalf("expected empty for missing file, got %q", got)
	}
}

func TestResolveTerminalFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// root.terminal
	os.WriteFile(configPath, []byte(`{"terminal":"alacritty"}`), 0644)
	if got := resolveTerminalFromConfig(configPath); got != "alacritty" {
		t.Fatalf("expected alacritty, got %q", got)
	}

	// appearance.terminal
	os.WriteFile(configPath, []byte(`{"appearance":{"terminal":"foot"}}`), 0644)
	if got := resolveTerminalFromConfig(configPath); got != "foot" {
		t.Fatalf("expected foot, got %q", got)
	}

	// root.terminal takes priority over appearance.terminal
	os.WriteFile(configPath, []byte(`{"terminal":"kitty","appearance":{"terminal":"foot"}}`), 0644)
	if got := resolveTerminalFromConfig(configPath); got != "kitty" {
		t.Fatalf("expected kitty, got %q", got)
	}

	// Missing
	os.WriteFile(configPath, []byte(`{}`), 0644)
	if got := resolveTerminalFromConfig(configPath); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestNoAutoStartFunctions(t *testing.T) {
	for _, fn := range []string{"close", "hide", "deactivate", "dismiss"} {
		if !noAutoStartFunctions[fn] {
			t.Fatalf("expected %q to be in noAutoStartFunctions", fn)
		}
	}
	if noAutoStartFunctions["toggle"] {
		t.Fatal("expected toggle to NOT be in noAutoStartFunctions")
	}
}