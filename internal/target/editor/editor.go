package editor

import (
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
	if !ctx.Config.WallpaperTheming.EnableZed {
		return nil
	}

	colors, err := ctx.ReadPaletteJSON()
	if err != nil {
		return err
	}

	pick := func(key, fallback string) string {
		if v, ok := colors[key]; ok && v != "" {
			return v
		}
		return fallback
	}

	_ = pick

	return nil
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
