package chrome

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

func TestNormalizeHex(t *testing.T) {
	t.Parallel()

	got, ok := normalizeHex(" #a1b2c3 ")
	if !ok || got != "#A1B2C3" {
		t.Fatalf("normalizeHex failed, got=%q ok=%v", got, ok)
	}

	if _, ok := normalizeHex("bad"); ok {
		t.Fatalf("normalizeHex should reject invalid values")
	}
}

func TestFixPreferencesSetsThemeValues(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	prefs := filepath.Join(tmp, "Preferences")
	if err := fixPreferences(prefs, "light"); err != nil {
		t.Fatalf("fixPreferences error: %v", err)
	}

	data, err := os.ReadFile(prefs)
	if err != nil {
		t.Fatalf("read prefs: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse prefs: %v", err)
	}

	browser := parsed["browser"].(map[string]interface{})
	theme := browser["theme"].(map[string]interface{})
	if theme["color_scheme"].(float64) != 1 {
		t.Fatalf("expected light mode color_scheme=1")
	}

	ext := parsed["extensions"].(map[string]interface{})
	extTheme := ext["theme"].(map[string]interface{})
	if extTheme["id"].(string) != "" {
		t.Fatalf("expected theme id to be reset")
	}
}

func TestApplyWritesPolicyAndPrefsForAvailableBrowser(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	outputDir := filepath.Join(tmp, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("mkdir outputDir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "color.txt"), []byte("#123abc"), 0644); err != nil {
		t.Fatalf("write color.txt: %v", err)
	}

	metaPath := filepath.Join(outputDir, "theme-meta.json")
	if err := os.WriteFile(metaPath, []byte(`{"mode":"light"}`), 0644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	policyDir := filepath.Join(tmp, "policies")
	prefsDir := filepath.Join(tmp, "prefs")

	originalLookPath := lookPath
	originalStartCommand := startCommand
	originalBrowserTargets := browserTargets
	t.Cleanup(func() {
		lookPath = originalLookPath
		startCommand = originalStartCommand
		browserTargets = originalBrowserTargets
	})

	lookPath = func(file string) (string, error) {
		if file == "chromium" {
			return "/usr/bin/chromium", nil
		}
		return "", errors.New("not found")
	}
	startCommand = func(name string, args ...string) error { return nil }
	browserTargets = func(_ *target.Context) []browserTarget {
		return []browserTarget{{bin: "chromium", policyDir: policyDir, prefsDir: prefsDir}}
	}

	ctx := &target.Context{
		Config: &config.Config{WallpaperTheming: config.WallpaperTheming{EnableChrome: true}},
		OutputDir: outputDir,
		MetaPath:  metaPath,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error: %v", err)
	}

	policyPath := filepath.Join(policyDir, "ii-theme.json")
	policyData, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("policy file not written: %v", err)
	}
	if !strings.Contains(string(policyData), "#123ABC") {
		t.Fatalf("policy should contain normalized seed color, got: %s", string(policyData))
	}

	prefsPath := filepath.Join(prefsDir, "Default", "Preferences")
	if _, err := os.Stat(prefsPath); err != nil {
		t.Fatalf("prefs file not written: %v", err)
	}
}
