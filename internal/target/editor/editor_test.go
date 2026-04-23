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
