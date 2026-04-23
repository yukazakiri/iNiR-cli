package steam

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowarch/inir-cli/internal/config"
	"github.com/snowarch/inir-cli/internal/target"
)

func TestGenerateSteamCSS(t *testing.T) {
	t.Parallel()

	css := generateSteamCSS(map[string]string{
		"primary":    "#112233",
		"on_primary": "#ffffff",
	})

	if !strings.Contains(css, "--adw-accent-bg-rgb: 17, 34, 51 !important;") {
		t.Fatalf("expected primary token mapping in css, got:\n%s", css)
	}
}

func TestRewriteLibraryRoot(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "libraryroot.custom.css")
	content := `@import "../adwaita/colorthemes/oldtheme/old.css";`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := rewriteLibraryRoot(path); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if !strings.Contains(string(updated), "colorthemes/inir/inir.css") {
		t.Fatalf("expected inir theme path replacement, got: %s", string(updated))
	}
}

func TestApplyDeploysToSteamDirs(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	outputDir := filepath.Join(tmp, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}

	cssPath := filepath.Join(outputDir, "steam-colortheme.css")
	if err := os.WriteFile(cssPath, []byte(":root { --adw-accent-bg-rgb: 1, 2, 3; }"), 0644); err != nil {
		t.Fatalf("write css: %v", err)
	}

	steamRoot := filepath.Join(home, ".steam", "steam")
	adwDir := filepath.Join(steamRoot, "steamui", "adwaita")
	if err := os.MkdirAll(filepath.Join(adwDir, "colorthemes"), 0755); err != nil {
		t.Fatalf("mkdir adw colorthemes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(steamRoot, "steamui", "libraryroot.custom.css"), []byte(`@import "../adwaita/colorthemes/old/old.css";`), 0644); err != nil {
		t.Fatalf("write libraryroot: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableAdwSteam: true},
		},
		OutputDir:   outputDir,
		PalettePath: filepath.Join(tmp, "missing-palette.json"),
		ColorsPath:  filepath.Join(tmp, "missing-colors.json"),
	}

	originalLookPath := lookPath
	originalRunCommand := runCommand
	originalIsProcessRunning := isProcessRunning
	t.Cleanup(func() {
		lookPath = originalLookPath
		runCommand = originalRunCommand
		isProcessRunning = originalIsProcessRunning
	})

	lookPath = func(file string) (string, error) {
		if file == "adwaita-steam-gtk" {
			return "/usr/bin/adwaita-steam-gtk", nil
		}
		return "", errors.New("not found")
	}
	runCommand = func(name string, args ...string) ([]byte, error) {
		return []byte("ok"), nil
	}
	isProcessRunning = func(name string) bool { return false }

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply returned error: %v", err)
	}

	themeCSS := filepath.Join(adwDir, "colorthemes", "inir", "inir.css")
	if _, err := os.Stat(themeCSS); err != nil {
		t.Fatalf("expected deployed theme css, stat error: %v", err)
	}

	customCSS := filepath.Join(adwDir, "custom", "custom.css")
	if _, err := os.Stat(customCSS); err != nil {
		t.Fatalf("expected deployed custom css, stat error: %v", err)
	}
}
