package sddm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/target"
)

type Applier struct{}

var sddmThemeDir = "/usr/share/sddm/themes/ii-pixel"

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("sddm apply: nil context or config")
	}

	if stat, err := os.Stat(sddmThemeDir); err != nil || !stat.IsDir() {
		return nil
	}

	colors, err := readSDDMColors(ctx)
	if err != nil {
		return nil
	}

	if err := updateThemeConf(filepath.Join(sddmThemeDir, "theme.conf"), colors); err != nil {
		return fmt.Errorf("update sddm theme.conf: %w", err)
	}

	if err := updateBackground(filepath.Join(sddmThemeDir, "assets", "background.png"), ctx.Config.Background.WallpaperPath); err != nil {
		fmt.Fprintf(os.Stderr, "[inir-cli] SDDM background sync skipped: %v\n", err)
	}

	return nil
}

func readSDDMColors(ctx *target.Context) (map[string]string, error) {
	palette, err := ctx.ReadPaletteJSON()
	if err != nil {
		palette, err = ctx.ReadColorsJSON()
		if err != nil {
			return nil, err
		}
	}

	pick := func(key, fallback string) string {
		if v := strings.TrimSpace(palette[key]); v != "" {
			return v
		}
		return fallback
	}

	return map[string]string{
		"primaryColor":          pick("primary", "#cba6f7"),
		"onPrimaryColor":        pick("on_primary", "#1e1e2e"),
		"surfaceColor":          pick("surface", "#1e1e2e"),
		"surfaceContainerColor": pick("surface_container", "#181825"),
		"onSurfaceColor":        pick("on_surface", "#cdd6f4"),
		"onSurfaceVariantColor": pick("on_surface_variant", "#9399b2"),
		"backgroundColor":       pick("background", "#1e1e2e"),
		"errorColor":            pick("error", "#f38ba8"),
	}, nil
}

func updateThemeConf(path string, values map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	remaining := map[string]string{}
	for key, value := range values {
		remaining[key] = value
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for key, value := range remaining {
			if strings.HasPrefix(trimmed, key+"=") {
				lines[i] = key + "=" + value
				delete(remaining, key)
				break
			}
		}
	}

	for key, value := range remaining {
		lines = append(lines, key+"="+value)
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(content), 0644)
}

func updateBackground(destinationPath, wallpaperPath string) error {
	path := strings.TrimSpace(strings.TrimPrefix(wallpaperPath, "file://"))
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destinationPath, data, 0644)
}
