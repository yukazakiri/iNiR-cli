package editor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowarch/inir-cli/internal/config"
	"github.com/snowarch/inir-cli/internal/target"
)

func TestGenerateZedThemeJSONContainsPrimaryColor(t *testing.T) {
	t.Parallel()

	data, err := generateZedThemeJSON(map[string]string{
		"primary":    "#112233",
		"on_surface": "#ddeeff",
	})
	if err != nil {
		t.Fatalf("generateZedThemeJSON error: %v", err)
	}

	if !strings.Contains(string(data), "#112233") {
		t.Fatalf("expected primary color in theme JSON, got:\n%s", string(data))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated theme JSON is invalid: %v", err)
	}
}

func TestZedApplierWritesThemeFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	palettePath := filepath.Join(tmp, "palette.json")
	if err := os.WriteFile(palettePath, []byte(`{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e"}`), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableZed: true},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
	}

	var a ZedApplier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("zed apply failed: %v", err)
	}

	themePath := filepath.Join(tmp, "config", "zed", "themes", "ii-theme.json")
	content, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("expected generated zed theme file: %v", err)
	}
	if !strings.Contains(string(content), "iNiR Dark") {
		t.Fatalf("expected zed theme name in generated file, got:\n%s", string(content))
	}
}

func TestEditorApplierWritesVSCodeCustomizations(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	palettePath := filepath.Join(tmp, "palette.json")
	terminalPath := filepath.Join(tmp, "terminal.json")
	if err := os.WriteFile(palettePath, []byte(`{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e","surface_container_low":"#181825","surface_container":"#313244"}`), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}
	if err := os.WriteFile(terminalPath, []byte(`{"term0":"#1e1e2e","term1":"#f38ba8","term2":"#a6e3a1","term3":"#f9e2af","term4":"#89b4fa","term5":"#cba6f7","term6":"#94e2d5","term7":"#cdd6f4"}`), 0644); err != nil {
		t.Fatalf("write terminal: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableVSCode: true},
		},
		PalettePath:  palettePath,
		ColorsPath:   palettePath,
		TerminalPath: terminalPath,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("editor apply failed: %v", err)
	}

	settingsPath := filepath.Join(tmp, "config", "Code", "User", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected vscode settings file: %v", err)
	}

	if !strings.Contains(string(content), "workbench.colorCustomizations") {
		t.Fatalf("settings missing workbench.colorCustomizations: %s", string(content))
	}
	if !strings.Contains(string(content), "terminal.ansiRed") {
		t.Fatalf("settings missing terminal palette customizations: %s", string(content))
	}
}
