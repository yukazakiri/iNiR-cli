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
	if !ctx.Config.WallpaperTheming.EnableVSCode {
		return nil
	}

	neovimEnabled := ctx.Config.WallpaperTheming.EnableNeovim
	if neovimEnabled {
		generateNeovimSpec(ctx)
	}

	return nil
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
