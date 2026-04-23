package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowarch/inir-cli/internal/config"
	"github.com/snowarch/inir-cli/internal/target"
)

func TestBuildANSISequencesContainsOSCEntries(t *testing.T) {
	t.Parallel()

	colors := map[string]string{}
	for i := 0; i <= 15; i++ {
		colors[fmt.Sprintf("term%d", i)] = "#112233"
	}
	colors["term0"] = "#101010"
	colors["term7"] = "#f0f0f0"

	sequences := buildANSISequences(colors)
	if !strings.Contains(sequences, "\x1b]4;0;#101010\x1b\\") {
		t.Fatalf("missing base palette sequence: %q", sequences)
	}
	if !strings.Contains(sequences, "\x1b]10;#f0f0f0\x1b\\") {
		t.Fatalf("missing foreground sequence: %q", sequences)
	}
}

func TestApplyWritesSequencesAndRespectsTerminalToggleMap(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	outputDir := filepath.Join(tmp, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}

	terminalPath := filepath.Join(outputDir, "terminal.json")
	terminalJSON := `{"term0":"#101010","term1":"#ff0000","term2":"#00ff00","term3":"#ffff00","term4":"#0000ff","term5":"#ff00ff","term6":"#00ffff","term7":"#f0f0f0","term8":"#202020","term9":"#ff1111","term10":"#11ff11","term11":"#ffff11","term12":"#1111ff","term13":"#ff11ff","term14":"#11ffff","term15":"#ffffff"}`
	if err := os.WriteFile(terminalPath, []byte(terminalJSON), 0644); err != nil {
		t.Fatalf("write terminal.json: %v", err)
	}

	palettePath := filepath.Join(outputDir, "palette.json")
	paletteJSON := `{"primary":"#8caaee","surface":"#1e1e2e","on_surface":"#dce0e8"}`
	if err := os.WriteFile(palettePath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("write palette.json: %v", err)
	}

	originalInject := injectAllPTS
	t.Cleanup(func() { injectAllPTS = originalInject })
	injectAllPTS = func(_ string) {}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{
				EnableTerminal: true,
				Terminals: map[string]bool{
					"kitty":     true,
					"alacritty": false,
					"wezterm":   false,
					"ghostty":   false,
					"foot":      false,
				},
			},
		},
		OutputDir:     outputDir,
		TerminalPath:  terminalPath,
		PalettePath:   palettePath,
		ColorsPath:    palettePath,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	sequencesPath := filepath.Join(outputDir, "terminal", "sequences.txt")
	if _, err := os.Stat(sequencesPath); err != nil {
		t.Fatalf("sequences.txt missing: %v", err)
	}

	kittyPath := filepath.Join(tmp, "config", "kitty", "current-theme.conf")
	if _, err := os.Stat(kittyPath); err != nil {
		t.Fatalf("kitty config missing: %v", err)
	}

	alacrittyPath := filepath.Join(tmp, "config", "alacritty", "colors.toml")
	if _, err := os.Stat(alacrittyPath); !os.IsNotExist(err) {
		t.Fatalf("alacritty config should not exist when disabled")
	}
}
