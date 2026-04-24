package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteChromiumThemeContractsWritesChromeContracts(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	err := writeChromiumThemeContracts(tmp, map[string]string{
		"primary":               "#1a2b3c",
		"surface_container_low": "#000000",
	})
	if err != nil {
		t.Fatalf("writeChromiumThemeContracts returned error: %v", err)
	}

	colorData, err := os.ReadFile(filepath.Join(tmp, "color.txt"))
	if err != nil {
		t.Fatalf("color.txt missing: %v", err)
	}
	if strings.TrimSpace(string(colorData)) != "#000000" {
		t.Fatalf("expected surface_container_low seed in color.txt, got %q", string(colorData))
	}

	chromeData, err := os.ReadFile(filepath.Join(tmp, "chromium.theme"))
	if err != nil {
		t.Fatalf("chromium.theme missing: %v", err)
	}
	if strings.TrimSpace(string(chromeData)) != "0,0,0" {
		t.Fatalf("expected RGB CSV in chromium.theme, got %q", string(chromeData))
	}
}

func TestPickChromiumThemeSeedFallsBackToSurface(t *testing.T) {
	t.Parallel()

	seed, ok := pickChromiumThemeSeed(map[string]string{
		"primary":    "invalid",
		"surface":    "#abcdef",
		"background": "#111111",
	})
	if !ok {
		t.Fatalf("expected fallback seed")
	}
	if seed != "#ABCDEF" {
		t.Fatalf("expected uppercase surface fallback, got %s", seed)
	}
}
