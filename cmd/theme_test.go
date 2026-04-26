package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGenerateForceDarkTerminalPreservesOriginalPalette(t *testing.T) {
	t.Parallel()

	cliPath := buildTestCLI(t)
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "generated")

	cmd := exec.Command(cliPath,
		"generate",
		"--color", "#FF6B35",
		"--mode", "light",
		"--scheme", "scheme-tonal-spot",
		"--force-dark-terminal",
		"--output", outDir,
	)
	cmd.Env = append(os.Environ(),
		"HOME="+tmp,
		"XDG_CONFIG_HOME="+filepath.Join(tmp, ".config"),
		"XDG_STATE_HOME="+filepath.Join(tmp, ".local", "state"),
		"XDG_CACHE_HOME="+filepath.Join(tmp, ".cache"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate failed: %v\noutput: %s", err, string(out))
	}

	// 1. palette.json must have light-mode background
	palette := readJSONMap(t, filepath.Join(outDir, "palette.json"))
	bgLightness := hexLuminance(palette["background"])
	if bgLightness < 0.5 {
		t.Fatalf("palette.json background should be light (mode=light), got %s with luminance %.2f", palette["background"], bgLightness)
	}

	// 2. terminal.json must have dark term0 (forced dark)
	terminal := readJSONMap(t, filepath.Join(outDir, "terminal.json"))
	term0Darkness := hexLuminance(terminal["term0"])
	if term0Darkness > 0.4 {
		t.Fatalf("terminal.json term0 should be dark (force-dark-terminal), got %s with luminance %.2f", terminal["term0"], term0Darkness)
	}

	// 3. chromium.theme must use the LIGHT palette (original result, not dark)
	chromiumData, err := os.ReadFile(filepath.Join(outDir, "chromium.theme"))
	if err != nil {
		t.Fatalf("chromium.theme missing: %v", err)
	}
	r, g, b, ok := parseChromiumTheme(string(chromiumData))
	if !ok {
		t.Fatalf("invalid chromium.theme content: %q", string(chromiumData))
	}
	avg := float64(r+g+b) / 3.0
	if avg < 128 {
		t.Fatalf("chromium.theme should use light palette (original mode), got avg RGB %.0f", avg)
	}

	// 4. material_colors.scss must be dark (overwritten by force-dark)
	scssData, err := os.ReadFile(filepath.Join(outDir, "material_colors.scss"))
	if err != nil {
		t.Fatalf("material_colors.scss missing: %v", err)
	}
	if !strings.Contains(string(scssData), "$darkmode: true") {
		t.Fatalf("material_colors.scss should have $darkmode: true after force-dark overwrite")
	}

	t.Logf("OK: light palette=%s, dark term0=%s, chromium RGB=(%d,%d,%d), scss darkmode=true",
		palette["background"], terminal["term0"], r, g, b)
}

func buildTestCLI(t *testing.T) string {
	t.Helper()
	tmpBin := filepath.Join(t.TempDir(), "inir-cli-test")
	cmd := exec.Command("go", "build", "-o", tmpBin, ".")
	cmd.Dir = "/home/admin/inir-cli"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build test CLI failed: %v\n%s", err, string(out))
	}
	return tmpBin
}

func readJSONMap(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return m
}

func hexLuminance(hex string) float64 {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return 0.5
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 32)
	g, _ := strconv.ParseInt(hex[2:4], 16, 32)
	b, _ := strconv.ParseInt(hex[4:6], 16, 32)
	// Simple perceived luminance
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
}

func TestGenerateWithTermScheme(t *testing.T) {
	t.Parallel()

	cliPath := buildTestCLI(t)
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "generated")

	// Create a custom termscheme with a distinctive custom base color
	schemePath := filepath.Join(tmp, "scheme-base.json")
	schemeData := map[string]map[string]string{
		"dark": {
			"term0": "#000000", "term1": "#FF0000", "term2": "#00FF00",
			"term3": "#0000FF", "term4": "#FFFF00", "term5": "#FF00FF",
			"term6": "#00FFFF", "term7": "#FFFFFF", "term8": "#808080",
			"term9": "#FF8080", "term10": "#80FF80", "term11": "#8080FF",
			"term12": "#FFFF80", "term13": "#FF80FF", "term14": "#80FFFF",
			"term15": "#C0C0C0",
		},
		"light": {
			"term0": "#FFFFFF", "term1": "#880000", "term2": "#008800",
			"term3": "#000088", "term4": "#888800", "term5": "#880088",
			"term6": "#008888", "term7": "#000000", "term8": "#808080",
			"term9": "#CC4444", "term10": "#44CC44", "term11": "#4444CC",
			"term12": "#CCCC44", "term13": "#CC44CC", "term14": "#44CCCC",
			"term15": "#404040",
		},
	}
	b, _ := json.MarshalIndent(schemeData, "", "  ")
	os.WriteFile(schemePath, b, 0644)

	cmd := exec.Command(cliPath,
		"generate",
		"--color", "#FF6B35",
		"--mode", "dark",
		"--scheme", "scheme-tonal-spot",
		"--termscheme", schemePath,
		"--output", outDir,
	)
	cmd.Env = append(os.Environ(),
		"HOME="+tmp,
		"XDG_CONFIG_HOME="+filepath.Join(tmp, ".config"),
		"XDG_STATE_HOME="+filepath.Join(tmp, ".local", "state"),
		"XDG_CACHE_HOME="+filepath.Join(tmp, ".cache"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate failed: %v\noutput: %s", err, string(out))
	}

	terminal := readJSONMap(t, filepath.Join(outDir, "terminal.json"))

	// With a custom termscheme, the base term1 should be harmonized from #FF0000,
	// not the default #FF6B6B. We just verify it's not the default.
	if terminal["term1"] == "#FF6B6B" {
		t.Fatalf("terminal.json term1 should use custom termscheme base, got default #FF6B6B")
	}

	// term0 is computed from surface, not the base scheme, so it will differ.
	// But term7 (which uses direct harmonization) should be derived from #FFFFFF (dark mode)
	// rather than the default #b0a99f.
	if terminal["term7"] == "#b0a99f" {
		t.Fatalf("terminal.json term7 should use custom termscheme base, got default #b0a99f")
	}

	t.Logf("OK: custom termscheme applied, term1=%s, term7=%s", terminal["term1"], terminal["term7"])
}

func parseChromiumTheme(s string) (int, int, int, bool) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	r, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	g, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	b, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}
