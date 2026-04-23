package editor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/snowarch/inir-cli/internal/target"
)

type Applier struct{}
type ZedApplier struct{}

func (a *Applier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("editor apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableVSCode {
		if ctx.Config.WallpaperTheming.EnableNeovim {
			generateNeovimSpec(ctx)
		}
		return nil
	}

	palette, err := ctx.ReadPaletteJSON()
	if err != nil {
		palette, err = ctx.ReadColorsJSON()
		if err != nil {
			return err
		}
	}

	terminal, err := ctx.ReadTerminalJSON()
	if err != nil {
		terminal = map[string]string{}
	}

	if err := applyVSCodeCustomizations(ctx, palette, terminal, ctx.Config.WallpaperTheming.VscodeEditors); err != nil {
		return err
	}

	neovimEnabled := ctx.Config.WallpaperTheming.EnableNeovim
	if neovimEnabled {
		generateNeovimSpec(ctx)
	}

	return nil
}

type vscodeFork struct {
	ConfigKey string
	DirName   string
}

func vscodeForks() []vscodeFork {
	return []vscodeFork{
		{ConfigKey: "code", DirName: "Code"},
		{ConfigKey: "codium", DirName: "VSCodium"},
		{ConfigKey: "codeOss", DirName: "Code - OSS"},
		{ConfigKey: "codeInsiders", DirName: "Code - Insiders"},
		{ConfigKey: "cursor", DirName: "Cursor"},
		{ConfigKey: "windsurf", DirName: "Windsurf"},
		{ConfigKey: "windsurfNext", DirName: "Windsurf - Next"},
		{ConfigKey: "qoder", DirName: "Qoder"},
		{ConfigKey: "antigravity", DirName: "Antigravity"},
		{ConfigKey: "positron", DirName: "Positron"},
		{ConfigKey: "voidEditor", DirName: "Void"},
		{ConfigKey: "melty", DirName: "Melty"},
		{ConfigKey: "pearai", DirName: "PearAI"},
		{ConfigKey: "aide", DirName: "Aide"},
	}
}

func applyVSCodeCustomizations(ctx *target.Context, palette map[string]string, terminal map[string]string, enabled map[string]bool) error {
	for _, fork := range vscodeForks() {
		if !isForkEnabled(enabled, fork.ConfigKey) {
			continue
		}

		settingsPath := filepath.Join(ctx.XDGConfigHome(), fork.DirName, "User", "settings.json")
		if err := updateVSCodeSettings(settingsPath, palette, terminal); err != nil {
			return err
		}
	}

	return nil
}

func isForkEnabled(enabled map[string]bool, key string) bool {
	if len(enabled) == 0 {
		return true
	}
	value, ok := enabled[key]
	if !ok {
		return true
	}
	return value
}

func updateVSCodeSettings(settingsPath string, palette map[string]string, terminal map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		data = []byte("{}")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		settings = map[string]interface{}{}
	}

	settings["workbench.colorTheme"] = "Default Dark+"
	settings["workbench.colorCustomizations"] = buildVSCodeColorCustomizations(palette, terminal)

	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, append(encoded, '\n'), 0644)
}

func buildVSCodeColorCustomizations(palette map[string]string, terminal map[string]string) map[string]interface{} {
	pickPalette := func(key, fallback string) string {
		if v, ok := palette[key]; ok && v != "" {
			return v
		}
		return fallback
	}
	pickTerminal := func(key, fallback string) string {
		if v, ok := terminal[key]; ok && v != "" {
			return v
		}
		return fallback
	}

	return map[string]interface{}{
		"editor.background":          pickPalette("surface_container_low", "#181825"),
		"editor.foreground":          pickPalette("on_surface", "#cdd6f4"),
		"editorCursor.foreground":    pickPalette("primary", "#cba6f7"),
		"activityBar.background":     pickPalette("surface", "#1e1e2e"),
		"sideBar.background":         pickPalette("surface", "#1e1e2e"),
		"statusBar.background":       pickPalette("surface_container", "#313244"),
		"statusBar.foreground":       pickPalette("on_surface", "#cdd6f4"),
		"titleBar.activeBackground":  pickPalette("surface_container", "#313244"),
		"titleBar.activeForeground":  pickPalette("on_surface", "#cdd6f4"),
		"terminal.background":        pickPalette("surface", "#1e1e2e"),
		"terminal.foreground":        pickPalette("on_surface", "#cdd6f4"),
		"terminal.ansiBlack":         pickTerminal("term0", "#1e1e2e"),
		"terminal.ansiRed":           pickTerminal("term1", "#f38ba8"),
		"terminal.ansiGreen":         pickTerminal("term2", "#a6e3a1"),
		"terminal.ansiYellow":        pickTerminal("term3", "#f9e2af"),
		"terminal.ansiBlue":          pickTerminal("term4", "#89b4fa"),
		"terminal.ansiMagenta":       pickTerminal("term5", "#cba6f7"),
		"terminal.ansiCyan":          pickTerminal("term6", "#94e2d5"),
		"terminal.ansiWhite":         pickTerminal("term7", "#cdd6f4"),
		"terminal.ansiBrightBlack":   pickTerminal("term8", "#585b70"),
		"terminal.ansiBrightRed":     pickTerminal("term9", "#f38ba8"),
		"terminal.ansiBrightGreen":   pickTerminal("term10", "#a6e3a1"),
		"terminal.ansiBrightYellow":  pickTerminal("term11", "#f9e2af"),
		"terminal.ansiBrightBlue":    pickTerminal("term12", "#89b4fa"),
		"terminal.ansiBrightMagenta": pickTerminal("term13", "#cba6f7"),
		"terminal.ansiBrightCyan":    pickTerminal("term14", "#94e2d5"),
		"terminal.ansiBrightWhite":   pickTerminal("term15", "#ffffff"),
	}
}

func (a *ZedApplier) Apply(ctx *target.Context) error {
	if ctx == nil || ctx.Config == nil {
		return fmt.Errorf("zed apply: nil context or config")
	}

	if !ctx.Config.WallpaperTheming.EnableZed {
		return nil
	}

	colors, err := ctx.ReadPaletteJSON()
	if err != nil {
		colors, err = ctx.ReadColorsJSON()
		if err != nil {
			return err
		}
	}

	themesDir := filepath.Join(ctx.XDGConfigHome(), "zed", "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(themesDir, "ii-theme.json")

	content, err := generateZedThemeJSON(colors)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return err
	}

	return nil
}

func generateZedThemeJSON(colors map[string]string) ([]byte, error) {
	pick := func(key, fallback string) string {
		if v, ok := colors[key]; ok && v != "" {
			return v
		}
		return fallback
	}

	theme := map[string]interface{}{
		"$schema": "https://zed.dev/schema/themes/v0.2.0.json",
		"name":    "iNiR",
		"author":  "inir-cli",
		"themes": []interface{}{
			map[string]interface{}{
				"name":       "iNiR Dark",
				"appearance": "dark",
				"style": map[string]interface{}{
					"background":            pick("surface", "#1e1e2e"),
					"surface.background":    pick("surface_container", "#313244"),
					"text":                  pick("on_surface", "#cdd6f4"),
					"text.muted":            pick("on_surface_variant", "#a6adc8"),
					"editor.background":     pick("surface_container_low", "#181825"),
					"editor.foreground":     pick("on_surface", "#cdd6f4"),
					"panel.background":      pick("surface", "#1e1e2e"),
					"tab.active_background": pick("surface_container_high", "#45475a"),
					"border.focused":        pick("primary", "#cba6f7"),
					"icon.accent":           pick("primary", "#cba6f7"),
				},
			},
		},
	}

	b, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func generateNeovimSpec(ctx *target.Context) {
	nvimDir := filepath.Join(ctx.XDGConfigHome(), "nvim", "lua", "plugins")
	os.MkdirAll(nvimDir, 0755)

	colorsFile := `-- inir color palette module (auto-generated)
local generated_dir = vim.fn.expand("~/.local/state/quickshell/user/generated")

local function read_json(path)
  local ok, lines = pcall(vim.fn.readfile, path)
  if not ok or not lines or vim.tbl_isempty(lines) then return {} end
  local ok_decode, decoded = pcall(vim.json.decode, table.concat(lines, "\n"))
  if not ok_decode or type(decoded) ~= "table" then return {} end
  return decoded
end

local M = {}

function M.load()
  local palette = read_json(generated_dir .. "/palette.json")
  if vim.tbl_isempty(palette) then palette = read_json(generated_dir .. "/colors.json") end
  local terminal = read_json(generated_dir .. "/terminal.json")

  local function pick(tbl, key, fallback)
    local value = tbl[key]
    return type(value) == "string" and value ~= "" and value or fallback
  end

  local fg = pick(palette, "on_background", "#E8E1DE")
  local term4 = pick(terminal, "term4", "#B19FB6")
  local term11 = pick(terminal, "term11", "#E2CBB5")

  return {
    bg = pick(palette, "background", "#151311"),
    dark_bg = pick(palette, "surface_container_low", "#1E1B19"),
    fg = fg,
    fg_dim = pick(palette, "on_surface_variant", "#CFC4BD"),
    muted = pick(palette, "outline", "#998F88"),
    red = pick(terminal, "term1", "#CA917F"),
    yellow = term11,
    accent = pick(palette, "primary", term4),
    green = pick(terminal, "term2", "#BBBB97"),
    cyan = pick(terminal, "term6", "#B5C8AA"),
    blue = term4,
    purple = pick(terminal, "term5", "#BF9EA4"),
    cursor = fg,
  }
end

return M
`
	os.WriteFile(filepath.Join(nvimDir, "inir_colors.lua"), []byte(colorsFile), 0644)

	fmt.Fprintf(os.Stderr, "[inir-cli] Generated Neovim color spec\n")
}
