package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/snowarch/inir-cli/internal/target"
	"github.com/snowarch/inir-cli/internal/target/shared/colorutil"
)

type Applier struct{}

type browserTarget struct {
	bin       string
	policyDir string
	prefsDir  string
}

var (
	lookPath = exec.LookPath
	startCommand = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		return cmd.Start()
	}
	browserTargets = func(ctx *target.Context) []browserTarget {
		return []browserTarget{
			{bin: "google-chrome-stable", policyDir: "/etc/opt/chrome/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/google-chrome")},
			{bin: "chromium", policyDir: "/etc/chromium/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/chromium")},
			{bin: "brave", policyDir: "/etc/brave/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/BraveSoftware/Brave-Browser")},
		}
	}
)

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("chrome apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableChrome {
		return nil
	}

	seedColor, err := resolveSeedColor(ctx)
	if err != nil {
		return err
	}
	if seedColor == "" {
		return nil
	}

	mode := resolveMode(ctx)
	policyJSON, err := buildPolicyJSON(seedColor)
	if err != nil {
		return err
	}

	for _, b := range browserTargets(ctx) {
		if _, err := lookPath(b.bin); err != nil {
			continue
		}

		_ = writePolicy(filepath.Join(b.policyDir, "ii-theme.json"), policyJSON)

		prefsFile := filepath.Join(b.prefsDir, "Default", "Preferences")
		_ = os.MkdirAll(filepath.Dir(prefsFile), 0755)
		_ = fixPreferences(prefsFile, mode)

		_ = startCommand(b.bin, "--refresh-platform-policy", "--no-startup-window")
	}

	return nil
}

func resolveSeedColor(ctx *target.Context) (string, error) {
	if data, err := os.ReadFile(filepath.Join(ctx.OutputDir, "color.txt")); err == nil {
		if normalized, ok := normalizeHex(string(data)); ok {
			return normalized, nil
		}
	}

	colors, err := ctx.ReadPaletteJSON()
	if err != nil {
		colors, err = ctx.ReadColorsJSON()
		if err != nil {
			return "", err
		}
	}

	if normalized, ok := normalizeHex(colors["primary"]); ok {
		return normalized, nil
	}

	return "", nil
}

func resolveMode(ctx *target.Context) string {
	meta, err := ctx.ReadMetaJSON()
	if err != nil {
		return "dark"
	}
	if mode, ok := meta["mode"].(string); ok {
		mode = strings.ToLower(strings.TrimSpace(mode))
		if mode == "light" {
			return "light"
		}
	}
	return "dark"
}

func buildPolicyJSON(seedColor string) ([]byte, error) {
	policy := map[string]string{
		"BrowserThemeColor":  seedColor,
		"BrowserColorScheme": "device",
	}
	return json.Marshal(policy)
}

func writePolicy(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func fixPreferences(prefsFile, mode string) error {
	data, err := os.ReadFile(prefsFile)
	if err != nil {
		data = []byte("{}")
	}

	var prefs map[string]interface{}
	if err := json.Unmarshal(data, &prefs); err != nil {
		prefs = map[string]interface{}{}
	}

	cs := float64(2)
	if mode == "light" {
		cs = 1
	}

	if prefs["browser"] == nil {
		prefs["browser"] = map[string]interface{}{}
	}
	browser := prefs["browser"].(map[string]interface{})
	if browser["theme"] == nil {
		browser["theme"] = map[string]interface{}{}
	}
	theme := browser["theme"].(map[string]interface{})
	theme["color_scheme"] = cs
	theme["color_scheme2"] = cs
	delete(theme, "user_color")
	delete(theme, "user_color2")

	if prefs["extensions"] == nil {
		prefs["extensions"] = map[string]interface{}{}
	}
	ext := prefs["extensions"].(map[string]interface{})
	if ext["theme"] == nil {
		ext["theme"] = map[string]interface{}{}
	}
	extTheme := ext["theme"].(map[string]interface{})
	extTheme["id"] = ""
	extTheme["use_system"] = false
	extTheme["use_custom"] = false

	out, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	return os.WriteFile(prefsFile, out, 0644)
}

func normalizeHex(value string) (string, bool) {
	return colorutil.NormalizeHexUpper(value)
}
