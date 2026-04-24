package spicetify

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yukazakiri/inir-cli/internal/target"
	"github.com/yukazakiri/inir-cli/internal/target/shared/colorutil"
)

type Applier struct{}

const (
	spicetifyThemeName       = "Inir"
	spicetifyColorSchemeName = "matugen"

	bridgeStartMarker = "/* === iNiR CSS variable bridge - auto-generated, do not edit === */"
	bridgeEndMarker   = "/* === end iNiR CSS variable bridge === */"

	playbackStartMarker = "/* === iNiR playback controls fix - auto-generated === */"
	playbackEndMarker   = "/* === end iNiR playback controls fix === */"

	sleekCSSURL = "https://raw.githubusercontent.com/spicetify/spicetify-themes/master/Sleek/user.css"
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
	httpGet = func(url string) (*http.Response, error) {
		return http.Get(url)
	}
	osWriteFile = os.WriteFile
	osReadFile  = os.ReadFile
	osMkdirAll  = os.MkdirAll
)

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("spicetify apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableSpicetify {
		return nil
	}

	spicetifyPath, err := lookPath("spicetify")
	if err != nil {
		return fmt.Errorf("spicetify not found in PATH: %w", err)
	}

	log := newLogger(ctx)
	log("spicetify found at: %s", spicetifyPath)

	palette, err := ctx.ReadPaletteJSON()
	if err != nil {
		palette, err = ctx.ReadColorsJSON()
		if err != nil {
			log("palette/colors JSON not found: %v", err)
			return fmt.Errorf("read spicetify colors: %w", err)
		}
	}

	colors := readSpicetifyColors(palette)
	configPath := resolveSpicetifyConfigPath(ctx)
	spicetifyRoot := filepath.Dir(configPath)
	themeDir := filepath.Join(spicetifyRoot, "Themes", spicetifyThemeName)

	if err := osMkdirAll(themeDir, 0755); err != nil {
		return fmt.Errorf("create spicetify theme dir: %w", err)
	}

	colorFile := filepath.Join(themeDir, "color.ini")
	userCSS := filepath.Join(themeDir, "user.css")
	watchLock := filepath.Join(ctx.XDGStateHome(), "quickshell", "user", "generated", "spicetify_watch.lock")

	// Download Sleek base CSS if missing
	downloadSleekCSS(userCSS, log)

	// Patch existing CSS for readability
	patchExistingUserCSS(userCSS)

	// Write user.css bridge FIRST so that when color.ini lands (last) and
	// triggers spicetify watch's file-change debounce, user.css is already
	// fully updated. Reversed order caused watch to reload Spotify from
	// color.ini before user.css was written — leaving bridge vars stale.
	bridgeBlock := renderBridgeCSS(colors)
	if err := upsertUserCSS(userCSS, bridgeBlock); err != nil {
		return fmt.Errorf("write spicetify user.css bridge: %w", err)
	}
	log("wrote user.css bridge")

	playbackBlock := renderPlaybackControlsFix(colors)
	if err := upsertPlaybackFix(userCSS, playbackBlock); err != nil {
		return fmt.Errorf("write spicetify user.css playback fix: %w", err)
	}
	log("wrote playback controls fix")

	// Write color.ini LAST to trigger watch reload
	if err := osWriteFile(colorFile, []byte(renderColorINI(colors)), 0644); err != nil {
		return fmt.Errorf("write spicetify color.ini: %w", err)
	}
	log("wrote color.ini")

	// Configure spicetify
	if out, err := runCommand("spicetify", "config", "inject_css", "1", "replace_colors", "1"); err != nil {
		log("spicetify config inject_css failed: %v (output: %s)", err, string(out))
	}
	if out, err := runCommand("spicetify", "config", "current_theme", spicetifyThemeName, "color_scheme", spicetifyColorSchemeName); err != nil {
		log("spicetify config current_theme failed: %v (output: %s)", err, string(out))
	}
	log("configured spicetify theme")

	applyThemeWithFallback(log)

	spotifyRunning := isProcessRunning("spotify")
	watchRunning := isWatchActive()
	log("spotify running: %v, watch running: %v", spotifyRunning, watchRunning)

	if watchRunning {
		log("Watch mode active - theme applied and colors updated")
		return nil
	}

	if !spotifyRunning {
		log("Spotify not running - theme applied for next launch")
		return nil
	}

	log("Spotify running without watch - theme applied, starting watch")
	startWatchMode(watchLock, log)

	return nil
}

func newLogger(ctx *target.Context) func(format string, args ...interface{}) {
	logFile := filepath.Join(ctx.XDGStateHome(), "quickshell", "user", "generated", "spicetify_theme.log")
	if err := osMkdirAll(filepath.Dir(logFile), 0755); err != nil {
		logFile = "/tmp/spicetify_theme.log"
	}

	return func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		timestamp := time.Now().Format("15:04:05")
		line := fmt.Sprintf("[%s] [spicetify] %s\n", timestamp, msg)
		_ = appendToFile(logFile, line)
	}
}

func applyThemeWithFallback(log func(format string, args ...interface{})) {
	if out, err := runCommand("spicetify", "apply", "-s"); err != nil {
		log("spicetify apply failed: %v (output: %s)", err, string(out))
		if out, err := runCommand("spicetify", "backup", "apply"); err != nil {
			log("spicetify backup apply failed: %v (output: %s)", err, string(out))
		} else {
			log("spicetify backup apply succeeded")
		}
	} else {
		log("spicetify apply succeeded")
	}
}

func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
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

func readSpicetifyColors(colors map[string]string) map[string]string {
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
		"primary":                pick("primary", "#8caaee"),
		"on_primary":             pick("on_primary", "#1e3a5f"),
		"on_primary_container":   pick("on_primary_container", "#dce0e8"),
		"on_surface":             pick("on_surface", "#dce0e8"),
		"on_surface_variant":     pick("on_surface_variant", "#a6adc8"),
		"surface":                pick("surface", "#1e1e2e"),
		"surface_variant":        pick("surface_variant", "#45475a"),
		"surface_container_low":  pick("surface_container_low", "#181825"),
		"surface_container":      pick("surface_container", "#313244"),
		"surface_container_high": pick("surface_container_high", "#45475a"),
		"surface_container_highest": pick("surface_container_highest", "#494d64"),
		"primary_container":      pick("primary_container", "#313244"),
		"secondary":              pick("secondary", "#89b4fa"),
		"secondary_container":    pick("secondary_container", "#3d4c6b"),
		"tertiary":               pick("tertiary", "#94e2d5"),
		"outline":                pick("outline", "#585b70"),
		"outline_variant":        pick("outline_variant", "#45475a"),
		"error":                  pick("error", "#f38ba8"),
		"shadow":                 pick("shadow", "#000000"),
	}
}

func downloadSleekCSS(cssFile string, log func(string, ...interface{})) {
	needsDownload := false
	if info, err := os.Stat(cssFile); err == nil {
		// File exists - check if it actually has Sleek CSS or just our bridge blocks
		if info.Size() < 5000 {
			needsDownload = true
			log("Existing user.css too small (%d bytes), likely missing Sleek base CSS — re-downloading", info.Size())
		} else {
			data, err := osReadFile(cssFile)
			if err != nil || !strings.Contains(string(data), ".main-rootlist") {
				needsDownload = true
				log("Existing user.css missing Sleek selectors — re-downloading")
			}
		}
	} else {
		needsDownload = true
	}

	if !needsDownload {
		return
	}

	log("Downloading base CSS from Sleek theme...")
	resp, err := httpGet(sleekCSSURL)
	if err != nil {
		log("Warning: Failed to download base CSS: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log("Warning: Failed to download base CSS: HTTP %d", resp.StatusCode)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log("Warning: Failed to read base CSS: %v", err)
		return
	}

	content := string(data)
	// Fix hard-to-read right-side playback controls
	content = strings.ReplaceAll(content, "rgba(var(--spice-rgb-selected-row),.7)", "var(--spice-subtext)")

	if err := osWriteFile(cssFile, []byte(content), 0644); err != nil {
		log("Warning: Failed to write base CSS: %v", err)
		return
	}
	log("Downloaded base CSS (%d bytes)", len(content))
}

func patchExistingUserCSS(cssFile string) {
	data, err := osReadFile(cssFile)
	if err != nil {
		return
	}
	content := string(data)
	patched := strings.ReplaceAll(content, "rgba(var(--spice-rgb-selected-row),.7)", "var(--spice-subtext)")
	if patched != content {
		_ = osWriteFile(cssFile, []byte(patched), 0644)
	}
}

func renderColorINI(colors map[string]string) string {
	strip := func(key string) string {
		value := colors[key]
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
		strip("on_surface"),
		strip("on_surface_variant"),
		strip("surface"),
		strip("surface_container_low"),
		strip("surface_container"),
		strip("surface_container_high"),
		strip("shadow"),
		strip("on_surface_variant"),
		strip("primary"),
		strip("secondary_container"),
		strip("outline_variant"),
		strip("surface_container_highest"),
		strip("tertiary"),
		strip("error"),
		strip("outline"),
	)
}

func renderBridgeCSS(colors map[string]string) string {
	mainSecondary := colors["surface_container"]
	mainElevated := colors["surface_container_high"]
	highlight := colors["surface_container_high"]
	highlightElevated := colors["surface_container_highest"]
	navActive := colors["primary_container"]
	navActiveText := colors["on_primary_container"]
	playbackBar := colors["on_surface_variant"]
	playButton := colors["primary"]
	playButtonActive := colors["secondary_container"]
	buttonSecondary := colors["on_surface_variant"]
	primary := colors["primary"]
	outlineVariant := colors["outline_variant"]

	return fmt.Sprintf(`%s
:root {
  --spice-main-secondary:      %s;
  --spice-main-elevated:       %s;
  --spice-highlight:           %s;
  --spice-highlight-elevated:  %s;
  --spice-nav-active:          %s;
  --spice-nav-active-text:     %s;
  --spice-playback-bar:        %s;
  --spice-play-button:         %s;
  --spice-play-button-active:  %s;
  --spice-button-secondary:    %s;
  --spice-hover:               rgba(%s, 0.10);
  --spice-active:              rgba(%s, 0.18);
  --spice-border:              %s;

  --spice-rgb-main:            %s;
  --spice-rgb-main-secondary:  %s;
  --spice-rgb-sidebar:         %s;
  --spice-rgb-selected-row:    %s;
  --spice-rgb-button:          %s;
  --spice-rgb-shadow:          %s;
  --spice-rgb-misc:            %s;
}
%s
`,
		bridgeStartMarker,
		mainSecondary,
		mainElevated,
		highlight,
		highlightElevated,
		navActive,
		navActiveText,
		playbackBar,
		playButton,
		playButtonActive,
		buttonSecondary,
		hexToRGB(primary),
		hexToRGB(primary),
		outlineVariant,
		hexToRGB(colors["surface"]),
		hexToRGB(mainSecondary),
		hexToRGB(colors["surface_container_low"]),
		hexToRGB(colors["on_surface_variant"]),
		hexToRGB(colors["primary"]),
		hexToRGB(colors["shadow"]),
		hexToRGB(colors["outline"]),
		bridgeEndMarker,
	)
}

func renderPlaybackControlsFix(colors map[string]string) string {
	playbackRGB := hexToRGB(colors["on_surface_variant"])

	return fmt.Sprintf(`%s
.main-playbackBar__slider,
.playback-bar__progress-time-elapsed,
.main-playbackBar__slider::before {
  --spice-rgb-selected-row: %s;
}

.control-button,
.main-connectToDevice-button {
  color: var(--spice-subtext) !important;
}

.control-button:hover,
.main-connectToDevice-button:hover {
  color: var(--spice-text) !important;
}

.progress-bar {
  --spice-rgb-selected-row: %s;
}

.progress-bar__bg {
  background-color: rgba(%s, 0.3) !important;
}
%s
`,
		playbackStartMarker,
		playbackRGB,
		playbackRGB,
		playbackRGB,
		playbackEndMarker,
	)
}

func upsertUserCSS(path, managedBlock string) error {
	current := ""
	data, err := osReadFile(path)
	if err == nil {
		current = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	updated := upsertManagedBlock(current, managedBlock, bridgeStartMarker, bridgeEndMarker)
	return osWriteFile(path, []byte(updated), 0644)
}

func upsertPlaybackFix(path, managedBlock string) error {
	current := ""
	data, err := osReadFile(path)
	if err == nil {
		current = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	updated := upsertManagedBlock(current, managedBlock, playbackStartMarker, playbackEndMarker)
	// Playback fix goes at the TOP (prepended) so it takes priority
	if strings.Contains(updated, playbackStartMarker) {
		// Already inserted by upsertManagedBlock (appended) — we need to move to top
		// Extract the block
		re := regexp.MustCompile(regexp.QuoteMeta(playbackStartMarker) + ".*?" + regexp.QuoteMeta(playbackEndMarker) + "\n?")
		block := re.FindString(updated)
		updated = re.ReplaceAllString(updated, "")
		updated = strings.TrimLeft(updated, "\n")
		if block != "" {
			updated = block + "\n" + updated
		}
	}
	return osWriteFile(path, []byte(updated), 0644)
}

func upsertManagedBlock(current, managedBlock, startMarker, endMarker string) string {
	pattern := regexp.QuoteMeta(startMarker) + `(?s:.*?)` + regexp.QuoteMeta(endMarker) + `\n?`
	re := regexp.MustCompile(pattern)
	current = re.ReplaceAllString(current, "")
	current = strings.TrimLeft(current, "\n")

	if strings.TrimSpace(current) == "" {
		return managedBlock + "\n"
	}
	if !strings.HasSuffix(current, "\n") {
		current += "\n"
	}

	return current + "\n" + managedBlock + "\n"
}

func startWatchMode(watchLock string, log func(string, ...interface{})) {
	log("Starting spicetify watch mode...")
	_ = startCommand("spicetify", "watch", "-s")
	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	if isWatchActive() {
		_ = osWriteFile(watchLock, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
		log("Watch mode started")
	} else {
		log("Failed to start watch mode")
		_ = os.Remove(watchLock)
	}
}

func hexToRGB(value string) string {
	result, ok := colorutil.HexToRGBCSV(value, false)
	if !ok {
		return "0,0,0"
	}
	return result
}
