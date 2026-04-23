package sddm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowarch/inir-cli/internal/config"
	"github.com/snowarch/inir-cli/internal/target"
)

func TestUpdateThemeConf(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	themeConf := filepath.Join(tmp, "theme.conf")
	input := "[General]\nprimaryColor=#000000\n"
	if err := os.WriteFile(themeConf, []byte(input), 0644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	values := map[string]string{
		"primaryColor":    "#111111",
		"backgroundColor": "#222222",
	}

	if err := updateThemeConf(themeConf, values); err != nil {
		t.Fatalf("update theme conf: %v", err)
	}

	out, err := os.ReadFile(themeConf)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	content := string(out)

	if !strings.Contains(content, "primaryColor=#111111") {
		t.Fatalf("expected primaryColor update, got:\n%s", content)
	}
	if !strings.Contains(content, "backgroundColor=#222222") {
		t.Fatalf("expected appended backgroundColor, got:\n%s", content)
	}
}

func TestApplyUpdatesThemeAndBackground(t *testing.T) {
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "ii-pixel")
	assetsDir := filepath.Join(themeDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}

	themeConf := filepath.Join(themeDir, "theme.conf")
	if err := os.WriteFile(themeConf, []byte("[General]\n"), 0644); err != nil {
		t.Fatalf("write theme conf: %v", err)
	}

	wallpaper := filepath.Join(tmp, "wallpaper.png")
	if err := os.WriteFile(wallpaper, []byte("png-data"), 0644); err != nil {
		t.Fatalf("write wallpaper: %v", err)
	}

	palettePath := filepath.Join(tmp, "palette.json")
	palette := `{"primary":"#abcdef","on_surface":"#fedcba"}`
	if err := os.WriteFile(palettePath, []byte(palette), 0644); err != nil {
		t.Fatalf("write palette: %v", err)
	}

	originalThemeDir := sddmThemeDir
	t.Cleanup(func() {
		sddmThemeDir = originalThemeDir
	})
	sddmThemeDir = themeDir

	ctx := &target.Context{
		Config: &config.Config{
			Background: config.Background{WallpaperPath: wallpaper},
		},
		PalettePath: palettePath,
		ColorsPath:  palettePath,
	}

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error: %v", err)
	}

	updatedConf, err := os.ReadFile(themeConf)
	if err != nil {
		t.Fatalf("read updated conf: %v", err)
	}
	if !strings.Contains(string(updatedConf), "primaryColor=#abcdef") {
		t.Fatalf("expected primaryColor in conf, got:\n%s", string(updatedConf))
	}

	bgPath := filepath.Join(assetsDir, "background.png")
	bgData, err := os.ReadFile(bgPath)
	if err != nil {
		t.Fatalf("read synced background: %v", err)
	}
	if string(bgData) != "png-data" {
		t.Fatalf("unexpected background data: %s", string(bgData))
	}
}
