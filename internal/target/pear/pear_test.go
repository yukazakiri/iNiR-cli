package pear

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowarch/inir-cli/internal/config"
	"github.com/snowarch/inir-cli/internal/target"
)

func TestGeneratePearCSSUsesPaletteTokens(t *testing.T) {
	t.Parallel()

	css := generatePearCSS(map[string]string{
		"primary":    "#112233",
		"on_surface": "#445566",
	})

	if !strings.Contains(css, "--ctp-accent: #112233;") {
		t.Fatalf("expected accent token in generated CSS, got:\n%s", css)
	}
	if !strings.Contains(css, "--ctp-text: #445566;") {
		t.Fatalf("expected text token in generated CSS, got:\n%s", css)
	}
}

func TestRegisterThemeInConfigDeduplicatesThemePath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "YouTube Music")
	cssPath := filepath.Join(tmp, "generated", pearGeneratedCSSName)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	initial := map[string]interface{}{
		"options": map[string]interface{}{
			"themes": []interface{}{cssPath},
		},
	}
	data, _ := json.Marshal(initial)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	if err := registerThemeInConfig(configDir, cssPath); err != nil {
		t.Fatalf("register theme failed: %v", err)
	}

	updatedData, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}

	var updated map[string]interface{}
	if err := json.Unmarshal(updatedData, &updated); err != nil {
		t.Fatalf("parse updated config: %v", err)
	}

	options := updated["options"].(map[string]interface{})
	themes := options["themes"].([]interface{})
	if len(themes) != 1 {
		t.Fatalf("expected deduplicated themes length 1, got %d", len(themes))
	}
}

func TestApplyWritesPearCSSAndConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	outputDir := filepath.Join(tmp, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}

	palettePath := filepath.Join(outputDir, "palette.json")
	paletteJSON := `{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e"}`
	if err := os.WriteFile(palettePath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	originalLookPath := lookPath
	originalSystemDesktopDir := systemDesktopDir
	t.Cleanup(func() {
		lookPath = originalLookPath
		systemDesktopDir = originalSystemDesktopDir
	})

	systemDesktopDir = filepath.Join(tmp, "system-apps")
	if err := os.MkdirAll(systemDesktopDir, 0755); err != nil {
		t.Fatalf("mkdir fake system desktop dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(systemDesktopDir, "pear-desktop.desktop"), []byte("Exec=pear-desktop\n"), 0644); err != nil {
		t.Fatalf("write fake desktop file: %v", err)
	}

	lookPath = func(file string) (string, error) {
		if file == "pear-desktop" {
			return "/usr/bin/pear-desktop", nil
		}
		return "", errors.New("not found")
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnablePearDesktop: true},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
		OutputDir:   outputDir,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	cssOut := filepath.Join(outputDir, pearGeneratedCSSName)
	if _, err := os.Stat(cssOut); err != nil {
		t.Fatalf("generated css missing: %v", err)
	}

	configFile := filepath.Join(tmp, "config", "YouTube Music", "config.json")
	configData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("config.json missing: %v", err)
	}
	if !strings.Contains(string(configData), cssOut) {
		t.Fatalf("config.json does not reference generated css: %s", string(configData))
	}

	userDesktop := filepath.Join(tmp, ".local", "share", "applications", "pear-desktop.desktop")
	userDesktopData, err := os.ReadFile(userDesktop)
	if err != nil {
		t.Fatalf("desktop override missing: %v", err)
	}
	if !strings.Contains(string(userDesktopData), "--remote-debugging-port=9223") {
		t.Fatalf("desktop override missing CDP flag: %s", string(userDesktopData))
	}
}
