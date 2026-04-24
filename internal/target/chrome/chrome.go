package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/target"
	"github.com/yukazakiri/inir-cli/internal/target/shared/colorutil"
)

type Applier struct{}

type browserTarget struct {
	bin       string
	policyDir string
	prefsDir  string
}

type chromeTheme struct {
	color       string
	policyColor string
	mode        string
	variant     string
}

var (
	lookPath     = exec.LookPath
	startCommand = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		return cmd.Start()
	}
	commandOutput = func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		return cmd.CombinedOutput()
	}
	browserTargets = func(ctx *target.Context) []browserTarget {
		return []browserTarget{
			{bin: "google-chrome-stable", policyDir: "/etc/opt/chrome/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/google-chrome")},
			{bin: "google-chrome", policyDir: "/etc/opt/chrome/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/google-chrome")},
			{bin: "chromium", policyDir: "/etc/chromium/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/chromium")},
			{bin: "chromium-browser", policyDir: "/etc/chromium/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/chromium")},
			{bin: "brave", policyDir: "/etc/brave/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/BraveSoftware/Brave-Browser")},
			{bin: "brave-browser", policyDir: "/etc/brave/policies/managed", prefsDir: filepath.Join(ctx.Home(), ".config/BraveSoftware/Brave-Browser")},
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

	theme, err := resolveChromeTheme(ctx)
	if err != nil {
		return err
	}
	if theme.color == "" {
		return nil
	}

	for _, b := range availableBrowserTargets(ctx) {
		omarchy := isOmarchyBrowser(b.bin)
		policyJSON, err := buildPolicyJSON(theme, !omarchy)
		if err != nil {
			continue
		}

		strategy := "policy-mode"
		if omarchy {
			strategy = "omarchy-cli"
		}
		fmt.Fprintf(os.Stderr, "[inir-cli] Chrome target %s: strategy=%s mode=%s variant=%s color=%s\n", b.bin, strategy, theme.mode, theme.variant, theme.effectiveColor())

		_ = writePolicy(filepath.Join(b.policyDir, "ii-theme.json"), policyJSON)

		prefsFile := filepath.Join(b.prefsDir, "Default", "Preferences")
		_ = os.MkdirAll(filepath.Dir(prefsFile), 0755)
		_ = fixPreferences(prefsFile, theme.mode)

		_ = refreshBrowserWithMode(b, theme, omarchy)
	}

	return nil
}

func resolveChromeTheme(ctx *target.Context) (chromeTheme, error) {
	color, err := resolveSeedColor(ctx)
	if err != nil {
		return chromeTheme{}, err
	}

	mode := resolveMode(ctx)
	return chromeTheme{
		color:       color,
		policyColor: color,
		mode:        mode,
		variant:     resolveVariant(ctx),
	}, nil
}

func availableBrowserTargets(ctx *target.Context) []browserTarget {
	seenPolicyDirs := map[string]struct{}{}
	available := []browserTarget{}

	for _, b := range browserTargets(ctx) {
		if _, err := lookPath(b.bin); err != nil {
			continue
		}
		if _, seen := seenPolicyDirs[b.policyDir]; seen {
			continue
		}
		seenPolicyDirs[b.policyDir] = struct{}{}
		available = append(available, b)
	}

	return available
}

func resolveSeedColor(ctx *target.Context) (string, error) {
	if data, err := os.ReadFile(filepath.Join(ctx.OutputDir, "chromium.theme")); err == nil {
		if hex, ok := rgbCSVToHex(string(data)); ok {
			return hex, nil
		}
	}

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

	for _, key := range []string{"surface_container_low", "surface", "background", "primary"} {
		if normalized, ok := normalizeHex(colors[key]); ok {
			return normalized, nil
		}
	}

	return "", nil
}

func rgbCSVToHex(value string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 3 {
		return "", false
	}

	rgb := make([]int, 3)
	for i, part := range parts {
		component, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || component < 0 || component > 255 {
			return "", false
		}
		rgb[i] = component
	}

	return fmt.Sprintf("#%02X%02X%02X", rgb[0], rgb[1], rgb[2]), true
}

func resolveMode(ctx *target.Context) string {
	meta, err := ctx.ReadMetaJSON()
	if err == nil {
		if mode, ok := meta["mode"].(string); ok {
			mode = strings.ToLower(strings.TrimSpace(mode))
			if mode == "light" {
				return "light"
			}
			if mode == "dark" {
				return "dark"
			}
		}
	}

	if mode, ok := resolveModeFromSCSS(ctx); ok {
		return mode
	}

	return "dark"
}

func resolveModeFromSCSS(ctx *target.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	scssPath := strings.TrimSpace(ctx.SCSSPath)
	if scssPath == "" {
		scssPath = filepath.Join(ctx.OutputDir, "material_colors.scss")
	}

	data, err := os.ReadFile(scssPath)
	if err != nil {
		return "", false
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "$darkmode:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "$darkmode:"), ";"))
		value = strings.ToLower(value)
		if value == "true" {
			return "dark", true
		}
		if value == "false" {
			return "light", true
		}
		return "", false
	}

	return "", false
}

func resolveVariant(ctx *target.Context) string {
	variant := ""
	if ctx != nil && ctx.Config != nil {
		variant = strings.TrimSpace(ctx.Config.Appearance.Palette.Type)
	}
	if variant == "" || variant == "auto" {
		if ctx != nil {
			if meta, err := ctx.ReadMetaJSON(); err == nil {
				if scheme, ok := meta["scheme"].(string); ok {
					variant = strings.TrimSpace(scheme)
				}
			}
		}
	}
	if variant == "" || variant == "auto" || strings.EqualFold(variant, "preset") {
		return "tonal_spot"
	}

	variant = strings.TrimPrefix(variant, "scheme-")
	variant = strings.ReplaceAll(variant, "-", "_")

	switch variant {
	case "tonal_spot", "neutral", "vibrant", "expressive":
		return variant
	default:
		return "tonal_spot"
	}
}

func buildPolicyJSON(theme chromeTheme, forceMode bool) ([]byte, error) {
	colorScheme := "device"
	if forceMode {
		colorScheme = policyMode(theme.mode)
	}

	policy := map[string]string{
		"BrowserThemeColor":  theme.effectiveColor(),
		"BrowserColorScheme": colorScheme,
	}
	return json.Marshal(policy)
}

func policyMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "light") {
		return "light"
	}
	return "dark"
}

func (theme chromeTheme) effectiveColor() string {
	if strings.TrimSpace(theme.policyColor) != "" {
		return theme.policyColor
	}
	return theme.color
}

func writePolicy(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func refreshBrowserWithMode(b browserTarget, theme chromeTheme, omarchy bool) error {
	if omarchy {
		rgb, _ := colorutil.HexToRGBCSV(theme.effectiveColor(), false)
		return startCommand(
			b.bin,
			"--no-startup-window",
			"--refresh-platform-policy",
			"--set-user-color="+rgb,
			"--set-color-scheme="+theme.mode,
			"--set-color-variant="+theme.variant,
		)
	}

	return startCommand(b.bin, "--refresh-platform-policy", "--no-startup-window")
}

func isOmarchyBrowser(bin string) bool {
	binPath, err := lookPath(bin)
	if err != nil || strings.TrimSpace(binPath) == "" {
		return false
	}
	if _, err := lookPath("pacman"); err != nil {
		return false
	}

	output, err := commandOutput("pacman", "-Qo", binPath)
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(output)), "omarchy")
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

	cs := preferenceColorSchemeForMode(mode)

	browser := ensureObject(prefs, "browser")
	theme := ensureObject(browser, "theme")
	theme["color_scheme"] = cs
	theme["color_scheme2"] = cs
	delete(theme, "user_color")
	delete(theme, "user_color2")

	ext := ensureObject(prefs, "extensions")
	extTheme := ensureObject(ext, "theme")
	extTheme["id"] = ""
	extTheme["use_system"] = false
	extTheme["use_custom"] = false

	out, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	return os.WriteFile(prefsFile, out, 0644)
}

func ensureObject(parent map[string]interface{}, key string) map[string]interface{} {
	if child, ok := parent[key].(map[string]interface{}); ok {
		return child
	}

	child := map[string]interface{}{}
	parent[key] = child
	return child
}

func preferenceColorSchemeForMode(mode string) float64 {
	if strings.EqualFold(strings.TrimSpace(mode), "dark") {
		return 2
	}
	return 1
}

func normalizeHex(value string) (string, bool) {
	return colorutil.NormalizeHexUpper(value)
}
