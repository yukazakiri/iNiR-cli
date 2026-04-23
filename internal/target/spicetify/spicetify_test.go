package spicetify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/config"
	"github.com/yukazakiri/inir-cli/internal/target"
)

func TestBuildSpicetifySchemeUsesMappingsAndFallbacks(t *testing.T) {
	t.Parallel()

	input := map[string]string{
		"primary":               "#ABCDEF",
		"on_surface":            "#112233",
		"surface":               "#202020",
		"surface_container_low": "#101010",
	}

	scheme := buildSpicetifyScheme(input)

	if got, want := scheme["button"], "#abcdef"; got != want {
		t.Fatalf("button mismatch: got %s want %s", got, want)
	}
	if got, want := scheme["text"], "#112233"; got != want {
		t.Fatalf("text mismatch: got %s want %s", got, want)
	}
	if got, want := scheme["shadow"], "#000000"; got != want {
		t.Fatalf("fallback mismatch: got %s want %s", got, want)
	}
}

func TestRenderColorINIStripsHash(t *testing.T) {
	t.Parallel()

	scheme := map[string]string{
		"text":               "#ffffff",
		"subtext":            "#bbbbbb",
		"main":               "#111111",
		"sidebar":            "#222222",
		"player":             "#333333",
		"card":               "#444444",
		"shadow":             "#000000",
		"selected-row":       "#999999",
		"button":             "#abcdef",
		"button-active":      "#345678",
		"button-disabled":    "#aaaaaa",
		"tab-active":         "#bbbbcc",
		"notification":       "#00ffaa",
		"notification-error": "#ff3344",
		"misc":               "#101010",
	}

	ini := renderColorINI(scheme)
	if !strings.Contains(ini, "button             = abcdef") {
		t.Fatalf("expected stripped hex in color.ini, got:\n%s", ini)
	}
	if strings.Contains(ini, "#abcdef") {
		t.Fatalf("expected no # in color.ini, got:\n%s", ini)
	}
}

func TestUpsertManagedBlockReplacesExisting(t *testing.T) {
	t.Parallel()

	existing := "base css\n" + bridgeStartMarker + "\nold\n" + bridgeEndMarker + "\n"
	updated := upsertManagedBlock(existing, bridgeStartMarker+"\nnew\n"+bridgeEndMarker)

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

func TestApplyWritesThemeFilesAndRunsCommands(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

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
	watchStarted := false
	startCommand = func(name string, args ...string) error {
		watchStarted = true
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	isWatchActive = func() bool { return false }
	isProcessRunning = func(name string) bool { return name == "spotify" }

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
	if !strings.Contains(string(data), bridgeStartMarker) {
		t.Fatalf("user.css missing managed bridge block")
	}

	if !watchStarted {
		t.Fatalf("expected watch mode to start when spotify is running")
	}

	joinedCalls := strings.Join(calls, "\n")
	if !strings.Contains(joinedCalls, "spicetify config current_theme Inir color_scheme matugen") {
		t.Fatalf("expected current_theme config call, got:\n%s", joinedCalls)
	}
}
