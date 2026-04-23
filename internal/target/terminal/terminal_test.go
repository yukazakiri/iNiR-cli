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
	originalLookPath := lookPath
	originalRunCommand := runCommand
	t.Cleanup(func() {
		injectAllPTS = originalInject
		lookPath = originalLookPath
		runCommand = originalRunCommand
	})
	injectAllPTS = func(_ string) {}
	lookPath = func(file string) (string, error) { return "", os.ErrNotExist }
	runCommand = func(name string, args ...string) error { return nil }

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

	kittyMainConf := filepath.Join(tmp, "config", "kitty", "kitty.conf")
	kittyMainData, err := os.ReadFile(kittyMainConf)
	if err != nil {
		t.Fatalf("kitty.conf missing: %v", err)
	}
	if !strings.Contains(string(kittyMainData), "include current-theme.conf") {
		t.Fatalf("kitty.conf missing include line: %s", string(kittyMainData))
	}
}

func TestApplyRuntimeHooksTriggersReloadCommands(t *testing.T) {
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

	alacrittyMain := filepath.Join(tmp, "config", "alacritty", "alacritty.toml")
	if err := os.MkdirAll(filepath.Dir(alacrittyMain), 0755); err != nil {
		t.Fatalf("mkdir alacritty dir: %v", err)
	}
	if err := os.WriteFile(alacrittyMain, []byte("# base\n"), 0644); err != nil {
		t.Fatalf("write alacritty.toml: %v", err)
	}

	originalInject := injectAllPTS
	originalLookPath := lookPath
	originalRunCommand := runCommand
	t.Cleanup(func() {
		injectAllPTS = originalInject
		lookPath = originalLookPath
		runCommand = originalRunCommand
	})
	injectAllPTS = func(_ string) {}

	lookPath = func(file string) (string, error) {
		switch file {
		case "kitty", "foot", "qdbus6":
			return "/usr/bin/" + file, nil
		default:
			return "", os.ErrNotExist
		}
	}

	var commandCalls []string
	runCommand = func(name string, args ...string) error {
		commandCalls = append(commandCalls, name+" "+strings.Join(args, " "))
		return nil
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{
				EnableTerminal: true,
				Terminals: map[string]bool{
					"kitty":     true,
					"alacritty": true,
					"foot":      true,
				},
			},
		},
		OutputDir:    outputDir,
		TerminalPath: terminalPath,
		PalettePath:  palettePath,
		ColorsPath:   palettePath,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	joined := strings.Join(commandCalls, "\n")
	if !strings.Contains(joined, "pkill --signal SIGUSR1 -x kitty") {
		t.Fatalf("expected kitty reload command, got: %s", joined)
	}
	if !strings.Contains(joined, "pkill -USR1 foot") {
		t.Fatalf("expected foot reload command, got: %s", joined)
	}
	if !strings.Contains(joined, "qdbus6 org.kde.konsole") {
		t.Fatalf("expected konsole refresh command, got: %s", joined)
	}

	alacrittyData, err := os.ReadFile(alacrittyMain)
	if err != nil {
		t.Fatalf("read alacritty.toml: %v", err)
	}
	if !strings.Contains(string(alacrittyData), `import = ["~/.config/alacritty/colors.toml"]`) {
		t.Fatalf("missing import line in alacritty.toml: %s", string(alacrittyData))
	}
}

func TestApplyWritesKonsoleSchemeWhenEnabled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))

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
	if err := os.WriteFile(palettePath, []byte(`{"primary":"#8caaee"}`), 0644); err != nil {
		t.Fatalf("write palette.json: %v", err)
	}

	originalInject := injectAllPTS
	originalLookPath := lookPath
	originalRunCommand := runCommand
	t.Cleanup(func() {
		injectAllPTS = originalInject
		lookPath = originalLookPath
		runCommand = originalRunCommand
	})
	injectAllPTS = func(_ string) {}
	lookPath = func(file string) (string, error) { return "", os.ErrNotExist }
	runCommand = func(name string, args ...string) error { return nil }

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{
				EnableTerminal: true,
				Terminals: map[string]bool{
					"kitty":     false,
					"alacritty": false,
					"wezterm":   false,
					"ghostty":   false,
					"foot":      false,
					"konsole":   true,
				},
			},
		},
		OutputDir:    outputDir,
		TerminalPath: terminalPath,
		PalettePath:  palettePath,
		ColorsPath:   palettePath,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	path := filepath.Join(tmp, "data", "konsole", "ii-auto.colorscheme")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("konsole colorscheme missing: %v", err)
	}

	if !strings.Contains(string(data), "[General]") {
		t.Fatalf("konsole colorscheme missing header: %s", string(data))
	}
	if !strings.Contains(string(data), "Color=16,16,16") {
		t.Fatalf("konsole colorscheme missing converted RGB entries: %s", string(data))
	}
}
