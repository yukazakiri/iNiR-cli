package chrome

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/config"
	"github.com/yukazakiri/inir-cli/internal/target"
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

func TestPreferenceColorSchemeForMode(t *testing.T) {
	t.Parallel()

	if got := preferenceColorSchemeForMode("dark"); got != 2 {
		t.Fatalf("expected dark mode to map to 2, got %v", got)
	}
	if got := preferenceColorSchemeForMode("light"); got != 1 {
		t.Fatalf("expected light mode to map to 1, got %v", got)
	}
}

func TestResolveSeedColorFollowsINiRPriority(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "chromium.theme"), []byte("17, 34, 51\n"), 0644); err != nil {
		t.Fatalf("write chromium.theme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "color.txt"), []byte("#abcdef"), 0644); err != nil {
		t.Fatalf("write color.txt: %v", err)
	}

	ctx := &target.Context{OutputDir: tmp}
	got, err := resolveSeedColor(ctx)
	if err != nil {
		t.Fatalf("resolveSeedColor returned error: %v", err)
	}
	if got != "#112233" {
		t.Fatalf("expected chromium.theme RGB contract to win, got %s", got)
	}
}

func TestResolveSeedColorPaletteFallbackOrder(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	palettePath := filepath.Join(tmp, "palette.json")
	if err := os.WriteFile(palettePath, []byte(`{"primary":"#111111","surface":"#222222","surface_container_low":"#333333"}`), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	ctx := &target.Context{OutputDir: tmp, PalettePath: palettePath}
	got, err := resolveSeedColor(ctx)
	if err != nil {
		t.Fatalf("resolveSeedColor returned error: %v", err)
	}
	if got != "#333333" {
		t.Fatalf("expected surface_container_low fallback, got %s", got)
	}
}

func TestResolveVariantMapsChromeSupportedSchemes(t *testing.T) {
	t.Parallel()

	ctx := &target.Context{
		Config: &config.Config{
			Appearance: config.Appearance{Palette: config.Palette{Type: "scheme-expressive"}},
		},
	}

	if got := resolveVariant(ctx); got != "expressive" {
		t.Fatalf("expected expressive variant, got %s", got)
	}
}

func TestResolveVariantFallsBackUnsupportedSchemesToTonalSpot(t *testing.T) {
	t.Parallel()

	ctx := &target.Context{
		Config: &config.Config{
			Appearance: config.Appearance{Palette: config.Palette{Type: "scheme-fruit-salad"}},
		},
	}

	if got := resolveVariant(ctx); got != "tonal_spot" {
		t.Fatalf("expected unsupported variant to map to tonal_spot, got %s", got)
	}
}

func TestResolveVariantUsesTonalSpotForPresetMetaScheme(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	metaPath := filepath.Join(tmp, "theme-meta.json")
	if err := os.WriteFile(metaPath, []byte(`{"scheme":"preset"}`), 0644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	ctx := &target.Context{
		Config:   &config.Config{},
		MetaPath: metaPath,
	}

	if got := resolveVariant(ctx); got != "tonal_spot" {
		t.Fatalf("expected preset scheme to map to tonal_spot, got %s", got)
	}
}

func TestBuildPolicyJSONUsesExactHexColorAndDeviceMode(t *testing.T) {
	t.Parallel()

	data, err := buildPolicyJSON(chromeTheme{color: "#A1B2C3", policyColor: "#A1B2C3", mode: "light"}, false)
	if err != nil {
		t.Fatalf("buildPolicyJSON returned error: %v", err)
	}

	var policy map[string]string
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatalf("parse policy JSON: %v", err)
	}
	if policy["BrowserThemeColor"] != "#A1B2C3" {
		t.Fatalf("expected exact hex BrowserThemeColor, got %q", policy["BrowserThemeColor"])
	}
	if policy["BrowserColorScheme"] != "device" {
		t.Fatalf("expected BrowserColorScheme=device when not forcing mode, got %q", policy["BrowserColorScheme"])
	}
}

func TestBuildPolicyJSONUsesExplicitModeWhenForced(t *testing.T) {
	t.Parallel()

	data, err := buildPolicyJSON(chromeTheme{color: "#112233", mode: "dark"}, true)
	if err != nil {
		t.Fatalf("buildPolicyJSON returned error: %v", err)
	}

	var policy map[string]string
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatalf("parse policy JSON: %v", err)
	}
	if policy["BrowserColorScheme"] != "dark" {
		t.Fatalf("expected BrowserColorScheme=dark when forcing mode, got %q", policy["BrowserColorScheme"])
	}
}

func TestResolveModeFallsBackToSCSSWhenMetaMissing(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	scssPath := filepath.Join(tmp, "material_colors.scss")
	if err := os.WriteFile(scssPath, []byte("$darkmode: false;\n"), 0644); err != nil {
		t.Fatalf("write scss: %v", err)
	}

	ctx := &target.Context{MetaPath: filepath.Join(tmp, "missing-meta.json"), SCSSPath: scssPath, OutputDir: tmp}
	if got := resolveMode(ctx); got != "light" {
		t.Fatalf("expected SCSS fallback light mode, got %s", got)
	}
}

func TestResolveModeUsesMetaWhenAvailable(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	metaPath := filepath.Join(tmp, "theme-meta.json")
	if err := os.WriteFile(metaPath, []byte(`{"mode":"dark"}`), 0644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	scssPath := filepath.Join(tmp, "material_colors.scss")
	if err := os.WriteFile(scssPath, []byte("$darkmode: false;\n"), 0644); err != nil {
		t.Fatalf("write scss: %v", err)
	}

	ctx := &target.Context{MetaPath: metaPath, SCSSPath: scssPath, OutputDir: tmp}
	if got := resolveMode(ctx); got != "dark" {
		t.Fatalf("expected meta mode to take precedence, got %s", got)
	}
}

func TestAvailableBrowserTargetsDeduplicatesByPolicyDir(t *testing.T) {
	originalLookPath := lookPath
	originalBrowserTargets := browserTargets
	t.Cleanup(func() {
		lookPath = originalLookPath
		browserTargets = originalBrowserTargets
	})

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	browserTargets = func(_ *target.Context) []browserTarget {
		return []browserTarget{
			{bin: "google-chrome-stable", policyDir: "/etc/opt/chrome/policies/managed", prefsDir: "chrome"},
			{bin: "google-chrome", policyDir: "/etc/opt/chrome/policies/managed", prefsDir: "chrome"},
			{bin: "chromium", policyDir: "/etc/chromium/policies/managed", prefsDir: "chromium"},
		}
	}

	available := availableBrowserTargets(&target.Context{})
	if len(available) != 2 {
		t.Fatalf("expected 2 deduped browser targets, got %d", len(available))
	}
	if available[0].bin != "google-chrome-stable" || available[1].bin != "chromium" {
		t.Fatalf("unexpected browser target order: %+v", available)
	}
}

func TestRefreshBrowserUsesOmarchyLiveFlags(t *testing.T) {
	originalLookPath := lookPath
	originalStartCommand := startCommand
	originalCommandOutput := commandOutput
	t.Cleanup(func() {
		lookPath = originalLookPath
		startCommand = originalStartCommand
		commandOutput = originalCommandOutput
	})

	lookPath = func(file string) (string, error) {
		switch file {
		case "chromium", "pacman":
			return "/usr/bin/" + file, nil
		default:
			return "", errors.New("not found")
		}
	}
	commandOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("/usr/bin/chromium is owned by omarchy-chromium 1.0.0"), nil
	}

	var capturedName string
	var capturedArgs []string
	startCommand = func(name string, args ...string) error {
		capturedName = name
		capturedArgs = append([]string{}, args...)
		return nil
	}

	err := refreshBrowserWithMode(browserTarget{bin: "chromium"}, chromeTheme{color: "#112233", policyColor: "#112233", mode: "dark", variant: "vibrant"}, true)
	if err != nil {
		t.Fatalf("refreshBrowser returned error: %v", err)
	}
	if capturedName != "chromium" {
		t.Fatalf("expected chromium command, got %s", capturedName)
	}
	joined := strings.Join(capturedArgs, " ")
	for _, want := range []string{"--set-user-color=17,34,51", "--set-color-scheme=dark", "--set-color-variant=vibrant"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected Omarchy arg %q in %q", want, joined)
		}
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
	originalCommandOutput := commandOutput
	originalBrowserTargets := browserTargets
	t.Cleanup(func() {
		lookPath = originalLookPath
		startCommand = originalStartCommand
		commandOutput = originalCommandOutput
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
		Config:    &config.Config{WallpaperTheming: config.WallpaperTheming{EnableChrome: true}},
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
	var policy map[string]string
	if err := json.Unmarshal(policyData, &policy); err != nil {
		t.Fatalf("parse policy JSON: %v", err)
	}
	if policy["BrowserColorScheme"] != "light" {
		t.Fatalf("policy should force BrowserColorScheme=light for standard browsers, got: %s", string(policyData))
	}
	if policy["BrowserThemeColor"] != "#123ABC" {
		t.Fatalf("policy should contain exact normalized seed color, got: %s", string(policyData))
	}

	prefsPath := filepath.Join(prefsDir, "Default", "Preferences")
	if _, err := os.Stat(prefsPath); err != nil {
		t.Fatalf("prefs file not written: %v", err)
	}
}
