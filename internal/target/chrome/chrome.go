package chrome

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/target"
)

type Applier struct{}

func (a *Applier) Apply(ctx *target.Context) error {
	if !ctx.Config.WallpaperTheming.EnableChrome {
		return nil
	}

	var seedColor string
	if data, err := os.ReadFile(filepath.Join(ctx.OutputDir, "color.txt")); err == nil {
		seedColor = strings.TrimSpace(string(data))
	}
	if seedColor == "" {
		colors, err := ctx.ReadPaletteJSON()
		if err != nil {
			return err
		}
		if v, ok := colors["primary"]; ok {
			seedColor = v
		}
	}
	if seedColor == "" {
		return nil
	}
	seedColor = strings.TrimPrefix(seedColor, "#")
	if len(seedColor) != 6 {
		return nil
	}
	seedColor = "#" + seedColor

	meta, _ := ctx.ReadMetaJSON()
	mode := "dark"
	if m, ok := meta["mode"].(string); ok {
		mode = m
	}

	variant := "tonal_spot"
	_ = variant
	if t, ok := meta["scheme"].(string); ok {
		t = strings.TrimPrefix(t, "scheme-")
		switch t {
		case "tonal-spot":
			variant = "tonal_spot"
		case "neutral":
			variant = "neutral"
		case "vibrant":
			variant = "vibrant"
		case "expressive":
			variant = "expressive"
		default:
			variant = "neutral"
		}
	}

	browsers := []struct {
		bin        string
		policyDir  string
		prefsDir   string
	}{
		{"google-chrome-stable", "/etc/opt/chrome/policies/managed", filepath.Join(ctx.Home(), ".config/google-chrome")},
		{"chromium", "/etc/chromium/policies/managed", filepath.Join(ctx.Home(), ".config/chromium")},
		{"brave", "/etc/brave/policies/managed", filepath.Join(ctx.Home(), ".config/BraveSoftware/Brave-Browser")},
	}

	for _, b := range browsers {
		if _, err := exec.LookPath(b.bin); err != nil {
			continue
		}

		policy := map[string]string{"BrowserThemeColor": seedColor}
		policyJSON, _ := json.Marshal(policy)
		policyDir := b.policyDir
		if err := os.MkdirAll(policyDir, 0755); err == nil {
			os.WriteFile(filepath.Join(policyDir, "ii-theme.json"), policyJSON, 0644)
		}

		prefsFile := filepath.Join(b.prefsDir, "Default", "Preferences")
		os.MkdirAll(filepath.Dir(prefsFile), 0755)
		fixPreferences(prefsFile, mode)

		exec.Command(b.bin, "--refresh-platform-policy", "--no-startup-window").Start()
	}

	return nil
}

func fixPreferences(prefsFile, mode string) {
	data, err := os.ReadFile(prefsFile)
	if err != nil {
		data = []byte("{}")
	}

	var prefs map[string]interface{}
	json.Unmarshal(data, &prefs)

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

	out, _ := json.Marshal(prefs)
	os.WriteFile(prefsFile, out, 0644)
}
