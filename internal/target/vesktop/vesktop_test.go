package vesktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/config"
	"github.com/yukazakiri/inir-cli/internal/target"
)

func TestGenerateVesktopCSSUsesPaletteTokens(t *testing.T) {
	t.Parallel()

	css := generateVesktopCSS(map[string]string{
		"primary":    "#112233",
		"on_surface": "#445566",
	})

	if !strings.Contains(css, "--inir-accent: #112233;") {
		t.Fatalf("expected accent token in generated CSS, got:\n%s", css)
	}
	if !strings.Contains(css, "--inir-fg: #445566;") {
		t.Fatalf("expected foreground token in generated CSS, got:\n%s", css)
	}
}

func TestResolveVesktopConfigDirPrefersExistingVesktopDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	vesktopDir := filepath.Join(tmp, "vesktop")
	if err := os.MkdirAll(vesktopDir, 0755); err != nil {
		t.Fatalf("mkdir vesktop dir: %v", err)
	}

	ctx := &target.Context{}
	resolved := resolveVesktopConfigDir(ctx)
	if resolved != vesktopDir {
		t.Fatalf("expected %s, got %s", vesktopDir, resolved)
	}
}

func TestApplyWritesThemeFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	outputDir := filepath.Join(tmp, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}

	palettePath := filepath.Join(outputDir, "palette.json")
	if err := os.WriteFile(palettePath, []byte(`{"primary":"#8caaee","on_surface":"#dce0e8","surface":"#1e1e2e"}`), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableVesktop: true},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
		OutputDir:   outputDir,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	themePath := filepath.Join(tmp, "config", "vesktop", "themes", vesktopThemeFileName)
	data, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("theme file missing: %v", err)
	}
	if !strings.Contains(string(data), "--inir-accent: #8caaee;") {
		t.Fatalf("theme file missing expected accent variable: %s", string(data))
	}
}
