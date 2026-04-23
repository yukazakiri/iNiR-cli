package steam

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yukazakiri/inir-cli/internal/target"
	"github.com/yukazakiri/inir-cli/internal/target/shared/colorutil"
)

type Applier struct{}

const steamThemeName = "inir"

var (
	lookPath = exec.LookPath
	runCommand = func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		return cmd.CombinedOutput()
	}
)

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("steam apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableAdwSteam {
		return nil
	}

	adwCmd, found := resolveAdwSteamCommand()
	if !found {
		return nil
	}

	cssPath, err := resolveSteamCSSSource(ctx)
	if err != nil {
		return err
	}

	steamDirs := steamInstallDirs(ctx)
	if !skinInstalled(steamDirs) {
		if err := bootstrapSkin(adwCmd); err != nil {
			return nil
		}
	}

	if _, err := deployCSS(ctx, cssPath, steamDirs); err != nil {
		return err
	}

	return nil
}

func resolveAdwSteamCommand() ([]string, bool) {
	if _, err := lookPath("adwaita-steam-gtk"); err == nil {
		return []string{"adwaita-steam-gtk"}, true
	}

	if _, err := lookPath("flatpak"); err == nil {
		output, err := runCommand("flatpak", "list", "--app")
		if err == nil && strings.Contains(string(output), "io.github.Foldex.AdwSteamGtk") {
			return []string{"flatpak", "run", "io.github.Foldex.AdwSteamGtk"}, true
		}
	}

	return nil, false
}

func bootstrapSkin(cmd []string) error {
	if len(cmd) == 0 {
		return fmt.Errorf("empty adwsteam command")
	}
	args := append(append([]string{}, cmd[1:]...), "-i")
	_, err := runCommand(cmd[0], args...)
	return err
}

func resolveSteamCSSSource(ctx *target.Context) (string, error) {
	cssPath := filepath.Join(ctx.OutputDir, "steam-colortheme.css")
	if _, err := os.Stat(cssPath); err == nil {
		return cssPath, nil
	}

	colors, err := ctx.ReadPaletteJSON()
	if err != nil {
		colors, err = ctx.ReadColorsJSON()
		if err != nil {
			return "", fmt.Errorf("read steam colors: %w", err)
		}
	}

	if err := os.MkdirAll(ctx.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("create steam output dir: %w", err)
	}

	if err := os.WriteFile(cssPath, []byte(generateSteamCSS(colors)), 0644); err != nil {
		return "", fmt.Errorf("write generated steam css: %w", err)
	}

	return cssPath, nil
}

func steamInstallDirs(ctx *target.Context) []string {
	home := ctx.Home()
	return []string{
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".steam", "steam"),
	}
}

func skinInstalled(steamDirs []string) bool {
	for _, dir := range steamDirs {
		if info, err := os.Stat(filepath.Join(dir, "steamui", "adwaita", "colorthemes")); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func deployCSS(ctx *target.Context, cssPath string, steamDirs []string) (int, error) {
	if err := copyFile(cssPath, filepath.Join(ctx.XDGConfigHome(), "AdwSteamGtk", "custom.css")); err != nil {
		return 0, fmt.Errorf("write AdwSteamGtk custom css: %w", err)
	}

	cacheThemePath := filepath.Join(resolveXDGCacheHome(ctx), "AdwSteamInstaller", "extracted", "adwaita", "colorthemes")
	if info, err := os.Stat(cacheThemePath); err == nil && info.IsDir() {
		if err := copyFile(cssPath, filepath.Join(cacheThemePath, steamThemeName, steamThemeName+".css")); err != nil {
			return 0, fmt.Errorf("write AdwSteamInstaller cache css: %w", err)
		}
	}

	deployed := 0
	for _, dir := range steamDirs {
		adwDir := filepath.Join(dir, "steamui", "adwaita")
		if info, err := os.Stat(adwDir); err != nil || !info.IsDir() {
			continue
		}

		themeCSS := filepath.Join(adwDir, "colorthemes", steamThemeName, steamThemeName+".css")
		if err := copyFile(cssPath, themeCSS); err != nil {
			return deployed, fmt.Errorf("write steam colortheme css: %w", err)
		}

		customCSS := filepath.Join(adwDir, "custom", "custom.css")
		if err := copyFile(cssPath, customCSS); err != nil {
			return deployed, fmt.Errorf("write steam custom css: %w", err)
		}

		if err := rewriteLibraryRoot(filepath.Join(dir, "steamui", "libraryroot.custom.css")); err != nil {
			return deployed, fmt.Errorf("rewrite steam libraryroot.custom.css: %w", err)
		}

		deployed++
	}

	return deployed, nil
}

func rewriteLibraryRoot(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, "colorthemes/"+steamThemeName+"/") {
		return nil
	}

	re := regexp.MustCompile(`colorthemes/[^/]*/[^"\s]+\.css`)
	updated := re.ReplaceAllString(content, "colorthemes/"+steamThemeName+"/"+steamThemeName+".css")
	if updated == content {
		return nil
	}

	return os.WriteFile(path, []byte(updated), 0644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func resolveXDGCacheHome(ctx *target.Context) string {
	if dir := os.Getenv("XDG_CACHE_HOME"); strings.TrimSpace(dir) != "" {
		return dir
	}
	return filepath.Join(ctx.Home(), ".cache")
}

func generateSteamCSS(colors map[string]string) string {
	readToken := func(name, fallback string) string {
		value, ok := colorutil.NormalizeHexLower(colors[name])
		if ok {
			return hexToRGBCSV(value)
		}
		if fallbackHex, ok := colorutil.NormalizeHexLower(fallback); ok {
			return hexToRGBCSV(fallbackHex)
		}
		return "0, 0, 0"
	}

	return fmt.Sprintf(`/**
 * iNiR Material You for Adwaita-for-Steam
 * Auto-generated from wallpaper colors. Do not edit.
 */
:root
{
	--adw-accent-bg-rgb: %s !important;
	--adw-accent-fg-rgb: %s !important;
	--adw-accent-rgb: %s !important;
	--adw-destructive-bg-rgb: %s !important;
	--adw-destructive-fg-rgb: %s !important;
	--adw-destructive-rgb: %s !important;
	--adw-success-bg-rgb: %s !important;
	--adw-success-fg-rgb: %s !important;
	--adw-success-rgb: %s !important;
	--adw-warning-bg-rgb: %s !important;
	--adw-warning-fg-rgb: %s !important;
	--adw-warning-rgb: %s !important;
	--adw-error-bg-rgb: %s !important;
	--adw-error-fg-rgb: %s !important;
	--adw-error-rgb: %s !important;
	--adw-window-bg-rgb: %s !important;
	--adw-window-fg-rgb: %s !important;
	--adw-view-bg-rgb: %s !important;
	--adw-view-fg-rgb: %s !important;
	--adw-headerbar-bg-rgb: %s !important;
	--adw-headerbar-fg-rgb: %s !important;
	--adw-headerbar-border-rgb: %s !important;
	--adw-headerbar-backdrop-rgb: %s !important;
	--adw-popover-bg-rgb: %s !important;
	--adw-popover-fg-rgb: %s !important;
	--adw-thumbnail-bg-rgb: %s !important;
	--adw-thumbnail-fg-rgb: %s !important;
	--adw-shade-rgb: %s !important;
	--adw-user-online-rgb: %s !important;
	--adw-user-ingame-rgb: %s !important;
}
`,
		readToken("primary", "#cba6f7"),
		readToken("on_primary", "#1e1e2e"),
		readToken("primary", "#cba6f7"),
		readToken("error", "#f38ba8"),
		readToken("on_error", "#ffffff"),
		readToken("error", "#f38ba8"),
		readToken("success", "#a6e3a1"),
		readToken("on_success", "#111111"),
		readToken("success", "#a6e3a1"),
		readToken("tertiary", "#fab387"),
		readToken("on_tertiary", "#111111"),
		readToken("tertiary", "#fab387"),
		readToken("error", "#f38ba8"),
		readToken("on_error", "#ffffff"),
		readToken("error", "#f38ba8"),
		readToken("surface_container_low", "#181825"),
		readToken("on_surface", "#cdd6f4"),
		readToken("surface", "#1e1e2e"),
		readToken("on_surface", "#cdd6f4"),
		readToken("surface_container", "#313244"),
		readToken("on_surface", "#cdd6f4"),
		readToken("outline_variant", "#6c7086"),
		readToken("surface_container_low", "#181825"),
		readToken("surface_container_high", "#45475a"),
		readToken("on_surface", "#cdd6f4"),
		readToken("surface_container_high", "#45475a"),
		readToken("on_surface", "#cdd6f4"),
		readToken("shadow", "#000000"),
		readToken("primary", "#cba6f7"),
		readToken("success", "#a6e3a1"),
	)
}

func hexToRGBCSV(value string) string {
	csv, ok := colorutil.HexToRGBCSV(value, true)
	if !ok {
		return "0, 0, 0"
	}
	return csv
}
