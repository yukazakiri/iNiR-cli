package steam

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukazakiri/inir-cli/internal/config"
	"github.com/yukazakiri/inir-cli/internal/target"
)

func TestGenerateSteamCSSUsesMaterialTokens(t *testing.T) {
	t.Parallel()

	css := generateSteamCSS(map[string]string{
		"primary": "#112233",
		"surface": "#445566",
	})

	if !strings.Contains(css, "--adw-accent-bg-rgb: 17, 34, 51 !important;") {
		t.Fatalf("expected primary token conversion in CSS, got:\n%s", css)
	}
	if !strings.Contains(css, "--adw-view-bg-rgb: 68, 85, 102 !important;") {
		t.Fatalf("expected surface token conversion in CSS, got:\n%s", css)
	}
}

func TestRewriteLibraryRootReplacesThemePath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "libraryroot.custom.css")
	content := `@import url("colorthemes/old-theme/old.css");`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	if err := rewriteLibraryRoot(file); err != nil {
		t.Fatalf("rewriteLibraryRoot error: %v", err)
	}

	updated, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(updated), "colorthemes/inir/inir.css") {
		t.Fatalf("expected theme path rewrite, got: %s", string(updated))
	}
}

func TestApplyDeploysSteamThemeFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	outputDir := filepath.Join(tmp, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}

	cssPath := filepath.Join(outputDir, "steam-colortheme.css")
	if err := os.WriteFile(cssPath, []byte(":root{}"), 0644); err != nil {
		t.Fatalf("write css source: %v", err)
	}

	steamDir := filepath.Join(tmp, ".steam", "steam", "steamui", "adwaita")
	if err := os.MkdirAll(filepath.Join(steamDir, "colorthemes"), 0755); err != nil {
		t.Fatalf("create steam adwaita dirs: %v", err)
	}

	libraryRoot := filepath.Join(tmp, ".steam", "steam", "steamui", "libraryroot.custom.css")
	if err := os.MkdirAll(filepath.Dir(libraryRoot), 0755); err != nil {
		t.Fatalf("create libraryroot parent: %v", err)
	}
	if err := os.WriteFile(libraryRoot, []byte(`@import url("colorthemes/foo/bar.css");`), 0644); err != nil {
		t.Fatalf("write libraryroot.custom.css: %v", err)
	}

	ctx := &target.Context{
		Config: &config.Config{
			WallpaperTheming: config.WallpaperTheming{EnableAdwSteam: true},
		},
		OutputDir: outputDir,
	}

	originalLookPath := lookPath
	originalRunCommand := runCommand
	t.Cleanup(func() {
		lookPath = originalLookPath
		runCommand = originalRunCommand
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

	var a Applier
	if err := a.Apply(ctx); err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(steamDir, "colorthemes", "inir", "inir.css")); err != nil {
		t.Fatalf("theme css not deployed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(steamDir, "custom", "custom.css")); err != nil {
		t.Fatalf("custom css not deployed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "config", "AdwSteamGtk", "custom.css")); err != nil {
		t.Fatalf("AdwSteamGtk custom css not deployed: %v", err)
	}

	updatedLibraryRoot, err := os.ReadFile(libraryRoot)
	if err != nil {
		t.Fatalf("read libraryroot.custom.css: %v", err)
	}
	if !strings.Contains(string(updatedLibraryRoot), "colorthemes/inir/inir.css") {
		t.Fatalf("libraryroot.custom.css was not rewritten: %s", string(updatedLibraryRoot))
	}
}
