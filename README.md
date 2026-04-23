# inir-cli

A Go CLI for generating Material You color palettes and applying them across 30+ desktop application targets. Extracted from the [iNiR](https://github.com/snowarch/iNiR) desktop shell as a standalone tool.

## Overview

`inir-cli` is the theming backbone of the iNiR desktop environment. It implements a two-stage pipeline:

1. **Palette Generation** — Generate Material Design 3 color schemes from wallpaper images or seed colors, using a pure Go implementation of HCT color space, QuantizeCelebi seed extraction, and 10 scheme variants.
2. **Target Application** — Apply generated colors across GTK, KDE, terminals, editors, browsers, music players, and more.

It can also apply any of 44 built-in static theme presets (Catppuccin, Gruvbox, Tokyo Night, Nord, etc.) without requiring a wallpaper.

## Installation

```bash
go build -o bin/inir-cli .
```

One-line local install (builds to `~/.local/bin/inir-cli` and makes it executable):

```bash
mkdir -p ~/.local/bin && go build -o ~/.local/bin/inir-cli . && chmod +x ~/.local/bin/inir-cli
```

Make sure `~/.local/bin` is on your `PATH`.

Requires Go 1.24+.

## Usage

### Generate colors from wallpaper

```bash
inir-cli generate --image /path/to/wallpaper.jpg --mode dark
inir-cli generate --color "#FF6B35" --scheme scheme-tonal-spot
```

### Apply a built-in theme preset

```bash
# List all 44 available themes
inir-cli scheme --list

# Apply a theme (writes colors.json, palette.json, terminal.json, theme-meta.json, material_colors.scss)
inir-cli scheme catppuccin-mocha

# Apply theme and apply to all desktop targets
inir-cli scheme tokyo-night --apply

# Pick and apply a random theme preset
inir-cli scheme --random --apply

# Apply to a custom output directory
inir-cli scheme nord --output ~/.local/state/quickshell/user/generated
```

### Full pipeline: generate + apply

```bash
inir-cli theme generate --image /path/to/wallpaper.jpg
inir-cli theme apply gtk-kde terminals editors chrome
inir-cli theme apply all
```

### Auto-detect best scheme variant

```bash
inir-cli auto-detect /path/to/wallpaper.jpg
# Outputs: scheme-tonal-spot, scheme-vibrant, scheme-expressive, etc.
```

### Configuration

Reads from `~/.config/illogical-impulse/config.json` by default (same config as iNiR shell). Override with `--config`.

## Commands

| Command | Description |
|---|---|
| `generate` | Generate color palette from wallpaper image or seed color |
| `scheme` | Apply a built-in static theme preset (44 themes), including `--random` |
| `theme generate` | Full pipeline: generate palette from wallpaper |
| `theme apply [targets...]` | Apply generated colors to specified targets |
| `auto-detect [image]` | Detect the best Material You scheme variant for an image |

## Output Contract

All generation commands write to the same file contract used by the iNiR shell:

| File | Description |
|---|---|
| `colors.json` | Full color set (material tokens + terminal 16-color palette) |
| `palette.json` | Material Design 3 semantic tokens only |
| `terminal.json` | Terminal 16-color ANSI palette (term0–term15) |
| `theme-meta.json` | Source metadata (preset/wallpaper, mode, scheme, terminal source) |
| `material_colors.scss` | SCSS variables for GTK theming |

Default output: `~/.local/state/quickshell/user/generated/`

## Architecture

```
inir-cli/
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # CLI root (cobra)
│   ├── theme.go                     # generate + apply commands
│   ├── scheme.go                    # scheme command + terminal harmonization
│   └── register.go                  # Target applier registration
├── internal/
│   ├── color/
│   │   ├── color.go                 # Re-exports
│   │   └── material/
│   │       ├── hct.go               # HCT color space (Hue, Chroma, Tone)
│   │       ├── lab.go               # CIE Lab/LCH conversions
│   │       ├── quantize.go          # Image quantization + seed scoring
│   │       ├── scheme.go            # 10 Material You scheme variants
│   │       ├── scheme_detect.go     # Auto-detect scheme from image
│   │       ├── terminal.go          # Terminal 16-color harmonization + WCAG
│   │       └── generate.go          # Master generator + JSON/SCSS output
│   ├── config/
│   │   └── config.go                # config.json loader
│   ├── presets/
│   │   ├── presets.go               # Preset types + helpers
│   │   └── data.go                  # 44 built-in theme definitions
│   ├── template/
│   │   └── template.go              # Template renderer ({{colors.TOKEN.MODE.hex}})
│   └── target/
│       ├── target.go                # Context, Applier interface, registry
│       ├── gtk/gtk.go               # GTK3/4 + KDE theme generation
│       ├── terminal/terminal.go     # ANSI sequence injection
│       ├── chrome/chrome.go         # Chrome/Chromium/Brave GM3 policy
│       ├── editor/editor.go         # Neovim + VS Code/Zed stubs
│       ├── spicetify/spicetify.go   # Spotify (stub)
│       ├── steam/steam.go           # Steam (stub)
│       ├── vesktop/vesktop.go       # Vesktop/Discord (stub)
│       ├── pear/pear.go             # YouTube Music desktop (stub)
│       └── sddm/sddm.go            # SDDM login screen (stub)
```

## Target Implementation Status

### Fully Implemented

| Target | Description |
|---|---|
| `terminals` | ANSI escape injection to `/dev/pts/*` |
| `chrome` | Chrome/Chromium/Brave GM3 BrowserThemeColor policy |

### Partially Implemented

| Target | Description | Issue |
|---|---|---|
| `gtk-kde` | GTK3/4 CSS, kdeglobals, Darkly.colors, qt5ct/qt6ct | `fmt.Sprintf` format string mismatches corrupt GTK4 CSS, GTK3 CSS, and Darkly.colors output |
| `editors` | Neovim color spec generation | VS Code theme generation is no-op; Zed reads palette but doesn't write |

### Stub (Not Yet Implemented)

| Target | Description | Reference Implementation |
|---|---|---|
| `spicetify` | Spotify via Spicetify + Sleek | `scripts/colors/apply-spicetify-theme.sh` (436 lines) |
| `steam` | Steam Adwaita-for-Steam CSS + CDP live injection | `scripts/colors/modules/70-steam.sh` (248 lines) |
| `vesktop` | Vesktop/Discord system24 palette | `scripts/colors/system24_palette.sh/py` |
| `pear-desktop` | YouTube Music Catppuccin CSS + CDP injection | `scripts/colors/modules/80-pear-desktop.sh` (1069+ lines) |
| `sddm` | SDDM login screen theme sync | `scripts/colors/modules/60-sddm.sh` |

## TODO

### High Priority

- [ ] **Fix GTK target** — resolve `fmt.Sprintf` format string mismatches in `generateGTK4CSS`, `generateGTK3CSS`, `generateDarklyColors`
- [ ] **VS Code theme generation** — port `scripts/colors/vscode/theme_generator.py` (1067 lines) to Go; existing `vscode_themegen` in the iNiR repo is a starting point
- [ ] **Zed theme generation** — port `scripts/colors/zed/theme_generator.py` (1029 lines) to Go; existing `zed_themegen` (1110 lines) in iNiR repo is a starting point
- [ ] **Terminal config generation** — port `scripts/colors/generate_terminal_configs.py` (1514 lines) for 11 terminal emulators (Kitty, Alacritty, WezTerm, Konsole, etc.)

### Medium Priority

- [ ] **Spicetify** — port `apply-spicetify-theme.sh` to Go
- [ ] **Steam** — port Steam CSS patching + CDP injection to Go
- [ ] **Vesktop/Discord** — port `system24_palette.py` to Go
- [ ] **Pear Desktop** — port YouTube Music theming + CDP to Go
- [ ] **SDDM** — port SDDM theme sync to Go

### Low Priority

- [ ] **Template rendering** — test and validate `{{colors.TOKEN.MODE.hex}}` template system against iNiR templates
- [ ] **Soften colors** — implement HSL saturation reduction for preset themes (currently only in QML)
- [ ] **OpenCode theme generator** — port `scripts/colors/opencode/theme_generator.py` (304 lines)
- [ ] **Image format support** — add WebP and AVIF decoding
- [ ] **Unit tests** — add tests for color science, scheme generation, terminal harmonization
- [ ] **Man page / shell completions** — generate from cobra

## Credits

Extracted from [iNiR](https://github.com/snowarch/iNiR) — a Hyprland-based desktop environment using Quickshell.

Color science based on Google's [Material Color Utilities](https://github.com/material-foundation/material-color-utilities) (HCT color space, QuantizeCelebi).

## License

Same as iNiR.
