package spicetify

import (
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
	spicetifyThemeName       = "Inir"
	spicetifyColorSchemeName = "matugen"

	bridgeStartMarker = "/* === iNiR CSS variable bridge - auto-generated, do not edit === */"
	bridgeEndMarker   = "/* === end iNiR CSS variable bridge === */"
)

var (
	lookPath = exec.LookPath
	runCommand = func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		return cmd.CombinedOutput()
	}
	startCommand = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		return cmd.Start()
	}
	isProcessRunning = func(name string) bool {
		return exec.Command("pgrep", "-x", name).Run() == nil
	}
	isWatchActive = func() bool {
		return exec.Command("pgrep", "-f", "spicetify watch").Run() == nil
	}
)

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("spicetify apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableSpicetify {
		return nil
	}

	if _, err := lookPath("spicetify"); err != nil {
		return nil
	}

	palette, err := ctx.ReadPaletteJSON()
	if err != nil {
		palette, err = ctx.ReadColorsJSON()
		if err != nil {
			return fmt.Errorf("read spicetify colors: %w", err)
		}
	}

	scheme := buildSpicetifyScheme(palette)
	configPath := resolveSpicetifyConfigPath(ctx)
	themeDir := filepath.Join(filepath.Dir(configPath), "Themes", spicetifyThemeName)

	if err := os.MkdirAll(themeDir, 0755); err != nil {
		return fmt.Errorf("create spicetify theme dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(themeDir, "color.ini"), []byte(renderColorINI(scheme)), 0644); err != nil {
		return fmt.Errorf("write spicetify color.ini: %w", err)
	}

	if err := upsertUserCSS(filepath.Join(themeDir, "user.css"), renderBridgeCSS(scheme)); err != nil {
		return fmt.Errorf("write spicetify user.css: %w", err)
	}

	_, _ = runCommand("spicetify", "config", "inject_css", "1", "replace_colors", "1")
	_, _ = runCommand("spicetify", "config", "current_theme", spicetifyThemeName, "color_scheme", spicetifyColorSchemeName)

	if isWatchActive() || !isProcessRunning("spotify") {
		return nil
	}

	_, _ = runCommand("spicetify", "refresh", "-s")
	_ = startCommand("spicetify", "watch", "-s")

	return nil
}

func resolveSpicetifyConfigPath(ctx *target.Context) string {
	defaultPath := filepath.Join(ctx.XDGConfigHome(), "spicetify", "config-xpui.ini")
	output, err := runCommand("spicetify", "-c")
	if err != nil {
		return defaultPath
	}

	configPath := strings.TrimSpace(string(output))
	if configPath == "" {
		return defaultPath
	}

	return configPath
}

func buildSpicetifyScheme(colors map[string]string) map[string]string {
	pick := func(key, fallback string) string {
		value := strings.TrimSpace(colors[key])
		if normalized, ok := colorutil.NormalizeHexLower(value); ok {
			return normalized
		}
		if normalized, ok := colorutil.NormalizeHexLower(fallback); ok {
			return normalized
		}
		return "#000000"
	}

	return map[string]string{
		"text":               pick("on_surface", "#dce0e8"),
		"subtext":            pick("on_surface_variant", "#a6adc8"),
		"main":               pick("surface", "#1e1e2e"),
		"sidebar":            pick("surface_container_low", "#181825"),
		"player":             pick("surface_container", "#313244"),
		"card":               pick("surface_container_high", "#45475a"),
		"shadow":             pick("shadow", "#000000"),
		"selected-row":       pick("on_surface_variant", "#a6adc8"),
		"button":             pick("primary", "#8caaee"),
		"button-active":      pick("secondary_container", "#3d4c6b"),
		"button-disabled":    pick("outline_variant", "#45475a"),
		"tab-active":         pick("surface_container_highest", "#494d64"),
		"notification":       pick("tertiary", "#94e2d5"),
		"notification-error": pick("error", "#f38ba8"),
		"misc":               pick("outline", "#585b70"),
	}
}

func renderColorINI(scheme map[string]string) string {
	toToken := func(key string) string {
		value := scheme[key]
		if value == "" {
			return "000000"
		}
		return strings.TrimPrefix(strings.ToLower(value), "#")
	}

	return fmt.Sprintf(`[matugen]
text               = %s
subtext            = %s
main               = %s
sidebar            = %s
player             = %s
card               = %s
shadow             = %s
selected-row       = %s
button             = %s
button-active      = %s
button-disabled    = %s
tab-active         = %s
notification       = %s
notification-error = %s
misc               = %s
`,
		toToken("text"),
		toToken("subtext"),
		toToken("main"),
		toToken("sidebar"),
		toToken("player"),
		toToken("card"),
		toToken("shadow"),
		toToken("selected-row"),
		toToken("button"),
		toToken("button-active"),
		toToken("button-disabled"),
		toToken("tab-active"),
		toToken("notification"),
		toToken("notification-error"),
		toToken("misc"),
	)
}

func renderBridgeCSS(scheme map[string]string) string {
	main := scheme["main"]
	sidebar := scheme["sidebar"]
	button := scheme["button"]
	text := scheme["text"]
	misc := scheme["misc"]

	buttonRGB := hexToRGB(button)
	mainRGB := hexToRGB(main)
	sidebarRGB := hexToRGB(sidebar)

	return fmt.Sprintf(`%s
:root {
  --spice-main-secondary: %s;
  --spice-main-elevated: %s;
  --spice-nav-active: %s;
  --spice-nav-active-text: %s;
  --spice-hover: rgba(%s, 0.10);
  --spice-active: rgba(%s, 0.18);
  --spice-border: %s;
  --spice-rgb-main: %s;
  --spice-rgb-main-secondary: %s;
  --spice-rgb-sidebar: %s;
}
%s
`,
		bridgeStartMarker,
		scheme["player"],
		scheme["card"],
		button,
		text,
		buttonRGB,
		buttonRGB,
		misc,
		mainRGB,
		mainRGB,
		sidebarRGB,
		bridgeEndMarker,
	)
}

func upsertUserCSS(path, managedBlock string) error {
	current := ""
	data, err := os.ReadFile(path)
	if err == nil {
		current = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	updated := upsertManagedBlock(current, managedBlock)
	return os.WriteFile(path, []byte(updated), 0644)
}

func upsertManagedBlock(current, managedBlock string) string {
	start := strings.Index(current, bridgeStartMarker)
	end := strings.Index(current, bridgeEndMarker)
	if start >= 0 && end >= start {
		end += len(bridgeEndMarker)
		for end < len(current) && (current[end] == '\n' || current[end] == '\r') {
			end++
		}
		current = current[:start] + current[end:]
	}

	current = strings.TrimLeft(current, "\n")
	if strings.TrimSpace(current) == "" {
		return managedBlock + "\n"
	}
	if !strings.HasSuffix(current, "\n") {
		current += "\n"
	}

	return current + "\n" + managedBlock + "\n"
}

func hexToRGB(value string) string {
	result, ok := colorutil.HexToRGBCSV(value, false)
	if !ok {
		return "0,0,0"
	}
	return result
}
