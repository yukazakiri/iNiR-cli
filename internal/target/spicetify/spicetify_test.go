package spicetify

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/config"
	"github.com/yukazakiri/inir-cli/internal/target"
)

func TestReadSpicetifyColorsUsesMappingsAndFallbacks(t *testing.T) {
	t.Parallel()

	input := map[string]string{
		"primary":                "#ABCDEF",
		"on_surface":             "#112233",
		"surface":                "#202020",
		"surface_container_low":  "#101010",
		"on_primary_container":   "#DDEEFF",
		"primary_container":      "#AABBCC",
		"outline_variant":        "#556677",
	}

	colors := readSpicetifyColors(input)

	if got, want := colors["primary"], "#abcdef"; got != want {
		t.Fatalf("primary mismatch: got %s want %s", got, want)
	}
	if got, want := colors["on_surface"], "#112233"; got != want {
		t.Fatalf("on_surface mismatch: got %s want %s", got, want)
	}
	if got, want := colors["on_primary_container"], "#ddeeff"; got != want {
		t.Fatalf("on_primary_container mismatch: got %s want %s", got, want)
	}
	if got, want := colors["primary_container"], "#aabbcc"; got != want {
		t.Fatalf("primary_container mismatch: got %s want %s", got, want)
	}
	if got, want := colors["outline_variant"], "#556677"; got != want {
		t.Fatalf("outline_variant mismatch: got %s want %s", got, want)
	}
	if got, want := colors["shadow"], "#000000"; got != want {
		t.Fatalf("fallback mismatch: got %s want %s", got, want)
	}
}

func TestRenderColorINIStripsHash(t *testing.T) {
	t.Parallel()

	colors := map[string]string{
		"on_surface":             "#ffffff",
		"on_surface_variant":     "#999999",
		"surface":                "#111111",
		"surface_container_low":  "#222222",
		"surface_container":      "#333333",
		"surface_container_high": "#444444",
		"shadow":                 "#000000",
		"primary":                "#abcdef",
		"secondary_container":    "#345678",
		"outline_variant":        "#aaaaaa",
		"surface_container_highest": "#bbbbcc",
		"tertiary":               "#00ffaa",
		"error":                  "#ff3344",
		"outline":                "#101010",
	}

	ini := renderColorINI(colors)
	if !strings.Contains(ini, "button             = abcdef") {
		t.Fatalf("expected stripped hex in color.ini, got:\n%s", ini)
	}
	if strings.Contains(ini, "#abcdef") {
		t.Fatalf("expected no # in color.ini, got:\n%s", ini)
	}
	if !strings.Contains(ini, "[matugen]") {
		t.Fatalf("expected [matugen] section, got:\n%s", ini)
	}
}

func TestRenderBridgeCSSHasCorrectVariables(t *testing.T) {
	t.Parallel()

	colors := map[string]string{
		"surface":                   "#1e1e2e",
		"surface_container":         "#313244",
		"surface_container_high":    "#45475a",
		"surface_container_highest": "#494d64",
		"surface_container_low":     "#181825",
		"primary_container":         "#313244",
		"on_primary_container":      "#dce0e8",
		"on_surface_variant":        "#a6adc8",
		"primary":                   "#8caaee",
		"secondary_container":       "#3d4c6b",
		"outline_variant":           "#45475a",
		"shadow":                    "#000000",
		"outline":                   "#585b70",
	}

	css := renderBridgeCSS(colors)

	// Verify all required variables are present with correct values
	checks := []struct {
		key   string
		value string
	}{
		{"--spice-main-secondary:", "#313244"},
		{"--spice-main-elevated:", "#45475a"},
		{"--spice-highlight:", "#45475a"},
		{"--spice-highlight-elevated:", "#494d64"},
		{"--spice-nav-active:", "#313244"},       // primary_container
		{"--spice-nav-active-text:", "#dce0e8"},  // on_primary_container
		{"--spice-playback-bar:", "#a6adc8"},     // on_surface_variant
		{"--spice-play-button:", "#8caaee"},      // primary
		{"--spice-play-button-active:", "#3d4c6b"}, // secondary_container
		{"--spice-button-secondary:", "#a6adc8"}, // on_surface_variant
		{"--spice-border:", "#45475a"},           // outline_variant
		{"--spice-hover:", "rgba(140,170,238, 0.10)"},
		{"--spice-active:", "rgba(140,170,238, 0.18)"},
		{"--spice-rgb-main:", "30,30,46"},
		{"--spice-rgb-main-secondary:", "49,50,68"},
		{"--spice-rgb-sidebar:", "24,24,37"},
		{"--spice-rgb-selected-row:", "166,173,200"},
		{"--spice-rgb-button:", "140,170,238"},
		{"--spice-rgb-shadow:", "0,0,0"},
		{"--spice-rgb-misc:", "88,91,112"},
	}

	for _, check := range checks {
		if !strings.Contains(css, check.key) {
			t.Fatalf("bridge CSS missing variable %s", check.key)
		}
		if !strings.Contains(css, check.value) {
			t.Fatalf("bridge CSS expected %s to contain %s, got:\n%s", check.key, check.value, css)
		}
	}
}

func TestRenderPlaybackControlsFix(t *testing.T) {
	t.Parallel()

	colors := map[string]string{
		"on_surface_variant": "#a6adc8",
	}

	css := renderPlaybackControlsFix(colors)

	if !strings.Contains(css, playbackStartMarker) {
		t.Fatalf("playback fix missing start marker")
	}
	if !strings.Contains(css, playbackEndMarker) {
		t.Fatalf("playback fix missing end marker")
	}
	if !strings.Contains(css, "--spice-rgb-selected-row: 166,173,200") {
		t.Fatalf("playback fix missing correct RGB value, got:\n%s", css)
	}
	if !strings.Contains(css, ".main-playbackBar__slider") {
		t.Fatalf("playback fix missing slider selector")
	}
	if !strings.Contains(css, ".control-button") {
		t.Fatalf("playback fix missing control-button selector")
	}
	if !strings.Contains(css, ".progress-bar__bg") {
		t.Fatalf("playback fix missing progress-bar__bg selector")
	}
}

func TestUpsertManagedBlockReplacesExisting(t *testing.T) {
	t.Parallel()

	existing := "base css\n" + bridgeStartMarker + "\nold\n" + bridgeEndMarker + "\n"
	updated := upsertManagedBlock(existing, bridgeStartMarker+"\nnew\n"+bridgeEndMarker, bridgeStartMarker, bridgeEndMarker)

	if strings.Count(updated, bridgeStartMarker) != 1 {
		t.Fatalf("expected one managed block, got:\n%s", updated)
	}
	if strings.Contains(updated, "old") {
		t.Fatalf("old managed block content still present:\n%s", updated)
	}
	if !strings.Contains(updated, "new") {
		t.Fatalf("new managed block content missing:\n%s", updated)
	}
}

func TestUpsertManagedBlockHandlesDuplicates(t *testing.T) {
	t.Parallel()

	existing := bridgeStartMarker + "\nold1\n" + bridgeEndMarker + "\n" +
		bridgeStartMarker + "\nold2\n" + bridgeEndMarker + "\n"
	updated := upsertManagedBlock(existing, bridgeStartMarker+"\nnew\n"+bridgeEndMarker, bridgeStartMarker, bridgeEndMarker)

	if strings.Count(updated, bridgeStartMarker) != 1 {
		t.Fatalf("expected exactly one managed block after dedup, got:\n%s", updated)
	}
}

func TestDownloadSleekCSSDownloadsAndPatches(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cssFile := filepath.Join(tmp, "user.css")

	called := false
	originalHTTPGet := httpGet
	httpGet = func(url string) (*http.Response, error) {
		called = true
		// Return mock CSS with the pattern that should be patched
		body := strings.NewReader(".test { color: rgba(var(--spice-rgb-selected-row),.7); }\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(body),
		}, nil
	}
	defer func() { httpGet = originalHTTPGet }()

	log := func(string, ...interface{}) {}
	downloadSleekCSS(cssFile, log)

	if !called {
		t.Fatalf("expected HTTP GET to be called")
	}

	data, err := os.ReadFile(cssFile)
	if err != nil {
		t.Fatalf("user.css not written: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "rgba(var(--spice-rgb-selected-row),.7)") {
		t.Fatalf("expected patched CSS, got:\n%s", content)
	}
	if !strings.Contains(content, "var(--spice-subtext)") {
		t.Fatalf("expected var(--spice-subtext) replacement, got:\n%s", content)
	}
}

func TestPatchExistingUserCSS(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cssFile := filepath.Join(tmp, "user.css")
	original := ".test { color: rgba(var(--spice-rgb-selected-row),.7); }\n"
	if err := os.WriteFile(cssFile, []byte(original), 0644); err != nil {
		t.Fatalf("write css: %v", err)
	}

	patchExistingUserCSS(cssFile)

	data, err := os.ReadFile(cssFile)
	if err != nil {
		t.Fatalf("read css: %v", err)
	}
	if strings.Contains(string(data), "rgba(var(--spice-rgb-selected-row),.7)") {
		t.Fatalf("expected patched CSS")
	}
}

func TestApplyWritesFilesInCorrectOrderAndStartsWatch(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))

	palettePath := filepath.Join(tmp, "palette.json")
	paletteJSON := `{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e","primary_container":"#313244","on_primary_container":"#dce0e8","on_surface_variant":"#a6adc8","surface_container":"#313244","surface_container_high":"#45475a","surface_container_highest":"#494d64","surface_container_low":"#181825","secondary_container":"#3d4c6b","outline_variant":"#45475a","outline":"#585b70","tertiary":"#94e2d5","error":"#f38ba8","shadow":"#000000"}`
	if err := os.WriteFile(palettePath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableSpicetify: true},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
	}

	originalLookPath := lookPath
	originalRunCommand := runCommand
	originalStartCommand := startCommand
	originalWatchActive := isWatchActive
	originalProcessRunning := isProcessRunning
	originalHTTPGet := httpGet
	t.Cleanup(func() {
		lookPath = originalLookPath
		runCommand = originalRunCommand
		startCommand = originalStartCommand
		isWatchActive = originalWatchActive
		isProcessRunning = originalProcessRunning
		httpGet = originalHTTPGet
	})

	var calls []string
	lookPath = func(file string) (string, error) {
		if file == "spicetify" {
			return "/usr/bin/spicetify", nil
		}
		return "", errors.New("not found")
	}
	runCommand = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		if len(args) > 0 && args[0] == "-c" {
			return []byte(filepath.Join(tmp, "spicetify", "config-xpui.ini")), nil
		}
		return []byte("ok"), nil
	}
	watchStarted := false
	startCommand = func(name string, args ...string) error {
		watchStarted = true
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	isWatchActive = func() bool { return false }
	isProcessRunning = func(name string) bool { return name == "spotify" }
	httpGet = func(url string) (*http.Response, error) {
		body := strings.NewReader(".sleek { }\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(body),
		}, nil
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error: %v", err)
	}

	colorINIPath := filepath.Join(tmp, "spicetify", "Themes", spicetifyThemeName, "color.ini")
	if _, err := os.Stat(colorINIPath); err != nil {
		t.Fatalf("color.ini not written: %v", err)
	}

	userCSSPath := filepath.Join(tmp, "spicetify", "Themes", spicetifyThemeName, "user.css")
	data, err := os.ReadFile(userCSSPath)
	if err != nil {
		t.Fatalf("user.css not written: %v", err)
	}
	cssContent := string(data)

	if !strings.Contains(cssContent, bridgeStartMarker) {
		t.Fatalf("user.css missing bridge block")
	}
	if !strings.Contains(cssContent, playbackStartMarker) {
		t.Fatalf("user.css missing playback controls fix")
	}

	// Verify bridge has correct nav-active color (primary_container, not primary)
	if !strings.Contains(cssContent, "--spice-nav-active:") {
		t.Fatalf("bridge CSS missing nav-active variable")
	}
	if !strings.Contains(cssContent, "#313244") {
		t.Fatalf("bridge CSS has wrong nav-active color. Expected primary_container (#313244)")
	}
	if !strings.Contains(cssContent, "--spice-nav-active-text:") {
		t.Fatalf("bridge CSS missing nav-active-text variable")
	}
	if !strings.Contains(cssContent, "#dce0e8") {
		t.Fatalf("bridge CSS has wrong nav-active-text color. Expected on_primary_container (#dce0e8)")
	}
	if !strings.Contains(cssContent, "--spice-border:") {
		t.Fatalf("bridge CSS missing border variable")
	}
	if !strings.Contains(cssContent, "#45475a") {
		t.Fatalf("bridge CSS has wrong border color. Expected outline_variant (#45475a)")
	}

	if !watchStarted {
		t.Fatalf("expected watch mode to start when spotify is running")
	}

	joinedCalls := strings.Join(calls, "\n")
	if !strings.Contains(joinedCalls, "spicetify config current_theme Inir color_scheme matugen") {
		t.Fatalf("expected current_theme config call, got:\n%s", joinedCalls)
	}
	if !strings.Contains(joinedCalls, "spicetify refresh -s") {
		t.Fatalf("expected refresh call, got:\n%s", joinedCalls)
	}
	if !strings.Contains(joinedCalls, "spicetify watch -s") {
		t.Fatalf("expected watch start call, got:\n%s", joinedCalls)
	}
}

func TestApplySkipsWhenWatchActive(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))

	palettePath := filepath.Join(tmp, "palette.json")
	paletteJSON := `{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e"}`
	if err := os.WriteFile(palettePath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableSpicetify: true},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
	}

	originalLookPath := lookPath
	originalRunCommand := runCommand
	originalStartCommand := startCommand
	originalWatchActive := isWatchActive
	originalProcessRunning := isProcessRunning
	t.Cleanup(func() {
		lookPath = originalLookPath
		runCommand = originalRunCommand
		startCommand = originalStartCommand
		isWatchActive = originalWatchActive
		isProcessRunning = originalProcessRunning
	})

	var calls []string
	lookPath = func(file string) (string, error) {
		if file == "spicetify" {
			return "/usr/bin/spicetify", nil
		}
		return "", errors.New("not found")
	}
	runCommand = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		if len(args) > 0 && args[0] == "-c" {
			return []byte(filepath.Join(tmp, "spicetify", "config-xpui.ini")), nil
		}
		return []byte("ok"), nil
	}
	startCommand = func(name string, args ...string) error {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	isWatchActive = func() bool { return true }
	isProcessRunning = func(name string) bool { return name == "spotify" }

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error: %v", err)
	}

	joinedCalls := strings.Join(calls, "\n")
	if strings.Contains(joinedCalls, "spicetify refresh") {
		t.Fatalf("expected no refresh when watch is active, got:\n%s", joinedCalls)
	}
	if strings.Contains(joinedCalls, "spicetify watch") {
		t.Fatalf("expected no watch start when watch is already active, got:\n%s", joinedCalls)
	}
}

func TestApplySkipsWhenSpotifyNotRunning(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))

	palettePath := filepath.Join(tmp, "palette.json")
	paletteJSON := `{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e"}`
	if err := os.WriteFile(palettePath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableSpicetify: true},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
	}

	originalLookPath := lookPath
	originalRunCommand := runCommand
	originalStartCommand := startCommand
	originalWatchActive := isWatchActive
	originalProcessRunning := isProcessRunning
	t.Cleanup(func() {
		lookPath = originalLookPath
		runCommand = originalRunCommand
		startCommand = originalStartCommand
		isWatchActive = originalWatchActive
		isProcessRunning = originalProcessRunning
	})

	var calls []string
	lookPath = func(file string) (string, error) {
		if file == "spicetify" {
			return "/usr/bin/spicetify", nil
		}
		return "", errors.New("not found")
	}
	runCommand = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		if len(args) > 0 && args[0] == "-c" {
			return []byte(filepath.Join(tmp, "spicetify", "config-xpui.ini")), nil
		}
		return []byte("ok"), nil
	}
	startCommand = func(name string, args ...string) error {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	isWatchActive = func() bool { return false }
	isProcessRunning = func(name string) bool { return false }

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error: %v", err)
	}

	joinedCalls := strings.Join(calls, "\n")
	if strings.Contains(joinedCalls, "spicetify refresh") {
		t.Fatalf("expected no refresh when spotify not running, got:\n%s", joinedCalls)
	}
	if strings.Contains(joinedCalls, "spicetify watch") {
		t.Fatalf("expected no watch start when spotify not running, got:\n%s", joinedCalls)
	}
}

func TestApplySkipsWhenDisabled(t *testing.T) {
	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableSpicetify: false},
		},
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error when disabled: %v", err)
	}
}

func TestApplySkipsWhenSpicetifyNotInstalled(t *testing.T) {
	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableSpicetify: true},
		},
	}

	originalLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() { lookPath = originalLookPath }()

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error when spicetify not installed: %v", err)
	}
}

func TestApplyReturnsErrorOnMissingColors(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableSpicetify: true},
		},
		PalettePath: filepath.Join(tmp, "nonexistent.json"),
		ColorsPath:  filepath.Join(tmp, "nonexistent2.json"),
	}

	originalLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "spicetify" {
			return "/usr/bin/spicetify", nil
		}
		return "", errors.New("not found")
	}
	defer func() { lookPath = originalLookPath }()

	var a Applier
	if err := a.Apply(ctx); err == nil {
		t.Fatalf("expected error when colors missing")
	}
}
