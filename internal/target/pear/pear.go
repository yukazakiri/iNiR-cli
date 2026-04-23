package pear

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/target"
	"github.com/yukazakiri/inir-cli/internal/target/shared/colorutil"
)

type Applier struct{}

const (
	pearGeneratedCSSName = "pear-desktop-theme.css"
	pearCDPPort          = "9223"
)

var (
	lookPath         = exec.LookPath
	systemDesktopDir = "/usr/share/applications"
)

type detectedPackage struct {
	binary string
}

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("pear apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnablePearDesktop {
		return nil
	}

	pkg := detectPackage(ctx)
	if pkg == nil {
		return nil
	}

	colors, err := readPearColors(ctx)
	if err != nil {
		return nil
	}

	cssPath := filepath.Join(ctx.OutputDir, pearGeneratedCSSName)
	if _, err := os.Stat(cssPath); err != nil {
		if err := os.MkdirAll(ctx.OutputDir, 0755); err != nil {
			return fmt.Errorf("create pear output dir: %w", err)
		}
		if err := os.WriteFile(cssPath, []byte(generatePearCSS(colors)), 0644); err != nil {
			return fmt.Errorf("write pear css: %w", err)
		}
	}

	configDir := filepath.Join(ctx.XDGConfigHome(), "YouTube Music")
	if err := registerThemeInConfig(configDir, cssPath); err != nil {
		return fmt.Errorf("register pear theme: %w", err)
	}

	if err := ensureDesktopOverride(ctx, pkg.binary, pearCDPPort); err != nil {
		fmt.Fprintf(os.Stderr, "[inir-cli] Pear desktop override skipped: %v\n", err)
	}

	return nil
}

func detectPackage(ctx *target.Context) *detectedPackage {
	for _, name := range []string{"pear-desktop", "youtube-music"} {
		if _, err := lookPath(name); err == nil {
			return &detectedPackage{binary: name}
		}
	}

	configDir := filepath.Join(ctx.XDGConfigHome(), "YouTube Music")
	if stat, err := os.Stat(configDir); err == nil && stat.IsDir() {
		return &detectedPackage{binary: "pear-desktop"}
	}

	return nil
}

func readPearColors(ctx *target.Context) (map[string]string, error) {
	palette, err := ctx.ReadPaletteJSON()
	if err != nil {
		palette, err = ctx.ReadColorsJSON()
		if err != nil {
			return nil, err
		}
	}
	return palette, nil
}

func generatePearCSS(colors map[string]string) string {
	pick := func(key, fallback string) string {
		if v, ok := colorutil.NormalizeHexLower(colors[key]); ok {
			return v
		}
		if v, ok := colorutil.NormalizeHexLower(fallback); ok {
			return v
		}
		return "#000000"
	}

	base := pick("surface", "#1e1e2e")
	mantle := pick("surface_container_low", "#181825")
	text := pick("on_surface", "#cdd6f4")
	subtext := pick("on_surface_variant", "#a6adc8")
	accent := pick("primary", "#cba6f7")
	errorColor := pick("error", "#f38ba8")

	return fmt.Sprintf(`/**
 * iNiR Material You — Pear Desktop (YouTube Music)
 * Auto-generated from wallpaper colors. Do not edit.
 */

html:not(.style-scope) {
  --ctp-base: %s;
  --ctp-mantle: %s;
  --ctp-text: %s;
  --ctp-subtext0: %s;
  --ctp-accent: %s;
  --ctp-red: %s;
}

body {
  background: var(--ctp-base) !important;
  color: var(--ctp-text) !important;
}

ytmusic-nav-bar {
  background: var(--ctp-mantle) !important;
}

a,
yt-formatted-string a {
  color: var(--ctp-accent) !important;
}

.error,
.ytp-error,
.warning {
  color: var(--ctp-red) !important;
}
`, base, mantle, text, subtext, accent, errorColor)
}

func registerThemeInConfig(configDir, cssPath string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		data = []byte("{}")
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		raw = map[string]interface{}{}
	}

	options, ok := raw["options"].(map[string]interface{})
	if !ok {
		options = map[string]interface{}{}
	}

	themeSet := map[string]struct{}{}
	themesRaw, ok := options["themes"].([]interface{})
	if ok {
		for _, item := range themesRaw {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				themeSet[s] = struct{}{}
			}
		}
	}
	themeSet[cssPath] = struct{}{}

	themes := make([]interface{}, 0, len(themeSet))
	for path := range themeSet {
		themes = append(themes, path)
	}

	options["themes"] = themes
	raw["options"] = options

	encoded, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, append(encoded, '\n'), 0644)
}

func ensureDesktopOverride(ctx *target.Context, binary, port string) error {
	if strings.TrimSpace(binary) == "" {
		return nil
	}

	systemDesktop := filepath.Join(systemDesktopDir, binary+".desktop")
	data, err := os.ReadFile(systemDesktop)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, "remote-debugging-port="+port) {
		return nil
	}

	needle := "Exec=" + binary
	replacement := "Exec=" + binary + " --remote-debugging-port=" + port
	if strings.Contains(content, replacement) {
		return nil
	}
	content = strings.Replace(content, needle, replacement, 1)

	userDesktop := filepath.Join(ctx.Home(), ".local", "share", "applications", binary+".desktop")
	if err := os.MkdirAll(filepath.Dir(userDesktop), 0755); err != nil {
		return err
	}

	return os.WriteFile(userDesktop, []byte(content), 0644)
}
