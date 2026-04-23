package steam

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/snowarch/inir-cli/internal/target"
)

type Applier struct{}

const steamThemeName = "inir"

var (
	lookPath = exec.LookPath
	runCommand = func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		return cmd.CombinedOutput()
	}
	isProcessRunning = func(name string) bool {
		return exec.Command("pgrep", "-x", name).Run() == nil
	}
)

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("steam apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableAdwSteam {
		return nil
	}

	adwCommand, ok := resolveAdwSteamCommand()
	if !ok {
		return nil
	}

	cssPath, err := resolveSteamCSS(ctx)
	if err != nil {
		return fmt.Errorf("resolve steam css: %w", err)
	}

	steamDirs := steamInstallDirs(ctx)
	if !skinInstalled(steamDirs) {
		if err := bootstrapSkin(adwCommand); err != nil {
			fmt.Fprintf(os.Stderr, "[inir-cli] steam bootstrap skipped: %v\n", err)
		}
	}

	if _, err := deployCSS(ctx, cssPath, steamDirs); err != nil {
		return fmt.Errorf("deploy steam css: %w", err)
	}

	if isProcessRunning("steamwebhelper") {
		fmt.Fprintf(os.Stderr, "[inir-cli] Steam running: CSS deployed, restart may be required for full refresh\n")
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

func bootstrapSkin(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("empty adwsteam command")
	}
	args := append([]string{}, command[1:]...)
	args = append(args, "-i")
	_, err := runCommand(command[0], args...)
	return err
}

func resolveSteamCSS(ctx *target.Context) (string, error) {
	cssPath := filepath.Join(ctx.OutputDir, "steam-colortheme.css")
	if _, err := os.Stat(cssPath); err == nil {
		return cssPath, nil
	}

	colors, err := ctx.ReadPaletteJSON()
	if err != nil {
		colors, err = ctx.ReadColorsJSON()
		if err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(ctx.OutputDir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(cssPath, []byte(generateSteamCSS(colors)), 0644); err != nil {
		return "", err
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
		if _, err := os.Stat(filepath.Join(dir, "steamui", "adwaita", "colorthemes")); err == nil {
			return true
		}
	}
	return false
}

func deployCSS(ctx *target.Context, cssPath string, steamDirs []string) (int, error) {
	xdgCache := os.Getenv("XDG_CACHE_HOME")
	if xdgCache == "" {
		xdgCache = filepath.Join(ctx.Home(), ".cache")
	}
	xdgConfig := ctx.XDGConfigHome()

	adwCacheDir := filepath.Join(xdgCache, "AdwSteamInstaller", "extracted", "adwaita", "colorthemes")
	if stat, err := os.Stat(adwCacheDir); err == nil && stat.IsDir() {
		if err := copyFile(cssPath, filepath.Join(adwCacheDir, steamThemeName, steamThemeName+".css")); err != nil {
			return 0, err
		}
	}

	if err := copyFile(cssPath, filepath.Join(xdgConfig, "AdwSteamGtk", "custom.css")); err != nil {
		return 0, err
	}

	deployed := 0
	for _, dir := range steamDirs {
		adwDir := filepath.Join(dir, "steamui", "adwaita")
		if stat, err := os.Stat(adwDir); err != nil || !stat.IsDir() {
			continue
		}

		if err := copyFile(cssPath, filepath.Join(adwDir, "colorthemes", steamThemeName, steamThemeName+".css")); err != nil {
			return deployed, err
		}
		if err := copyFile(cssPath, filepath.Join(adwDir, "custom", "custom.css")); err != nil {
			return deployed, err
		}

		_ = rewriteLibraryRoot(filepath.Join(dir, "steamui", "libraryroot.custom.css"))
		deployed++
	}

	return deployed, nil
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

func rewriteLibraryRoot(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)
	if strings.Contains(content, "colorthemes/"+steamThemeName+"/") {
		return nil
	}

	re := regexp.MustCompile(`colorthemes/[^/]+/[^"\n]+\.css`)
	replaced := re.ReplaceAllString(content, "colorthemes/"+steamThemeName+"/"+steamThemeName+".css")
	if replaced == content {
		return nil
	}

	return os.WriteFile(path, []byte(replaced), 0644)
}

func generateSteamCSS(colors map[string]string) string {
	getRGB := func(key, fallback string) string {
		if value, ok := normalizeHex(colors[key]); ok {
			return hexToRGB(value)
		}
		if value, ok := normalizeHex(fallback); ok {
			return hexToRGB(value)
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
	--adw-window-bg-rgb: %s !important;
	--adw-window-fg-rgb: %s !important;
	--adw-view-bg-rgb: %s !important;
	--adw-view-fg-rgb: %s !important;
	--adw-headerbar-bg-rgb: %s !important;
	--adw-headerbar-fg-rgb: %s !important;
	--adw-headerbar-border-rgb: %s !important;
	--adw-card-fg-rgb: %s !important;
	--adw-user-online-rgb: %s !important;
}
`,
		getRGB("primary", "#8caaee"),
		getRGB("on_primary", "#1e3a5f"),
		getRGB("primary", "#8caaee"),
		getRGB("error", "#f38ba8"),
		getRGB("on_error", "#ffffff"),
		getRGB("error", "#f38ba8"),
		getRGB("surface_container_low", "#181825"),
		getRGB("on_surface", "#dce0e8"),
		getRGB("surface", "#1e1e2e"),
		getRGB("on_surface", "#dce0e8"),
		getRGB("surface_container", "#313244"),
		getRGB("on_surface", "#dce0e8"),
		getRGB("outline_variant", "#45475a"),
		getRGB("on_surface", "#dce0e8"),
		getRGB("primary", "#8caaee"),
	)
}

func normalizeHex(value string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(value, "#"))
	if len(trimmed) != 6 {
		return "", false
	}
	if _, err := strconv.ParseUint(trimmed, 16, 32); err != nil {
		return "", false
	}
	return "#" + strings.ToLower(trimmed), true
}

func hexToRGB(value string) string {
	hex := strings.TrimPrefix(value, "#")
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return fmt.Sprintf("%d, %d, %d", r, g, b)
}
