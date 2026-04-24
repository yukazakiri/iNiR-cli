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
| `scheme` | Apply a built-in static theme preset (44 themes), including `--random`; with `--apply`, writes compatibility files and applies targets with progress output |
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
| `color.txt` | Compatibility seed color used by iNiR-style browser/theming scripts |
| `chromium.theme` | Browser seed RGB contract for Chromium-based theming (derived from `surface_container_low`, then `surface`, then `background`) |

`chromium.theme` is the source of truth for Chromium-based browser theme color application.

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
│       ├── chrome/chrome.go         # Chromium browser GM3 policy + live refresh
│       ├── editor/editor.go         # Neovim + VS Code/Zed/OpenCode theming
│       ├── spicetify/spicetify.go   # Spicetify theme + CSS bridge + refresh/watch
│       ├── steam/steam.go           # Adwaita-for-Steam CSS generation + deployment
│       ├── vesktop/vesktop.go       # Vesktop theme CSS generation
│       ├── pear/pear.go             # YouTube Music desktop CSS + config registration
│       └── sddm/sddm.go             # SDDM color + wallpaper synchronization
```

## Target Implementation Status

### Implemented (Current)

| Target | Description |
|---|---|
| `terminals` | ANSI injection + generated configs for Kitty, Alacritty, WezTerm, Ghostty, Foot, and Konsole + runtime reload hooks |
| `chrome` | Dual Chrome theming path: Omarchy forks use instant RGB CLI switches, official Chrome/Chromium/Brave use managed `BrowserThemeColor` policy + explicit mode sync, both consuming `chromium.theme` as the browser-color contract source of truth |
| `editors` | VS Code family color customizations, Zed theme file generation, OpenCode theme JSON generation, Neovim spec generation |
| `spicetify` | Spicetify theme generation (`color.ini` + CSS bridge), config wiring, refresh/watch trigger |
| `steam` | Adwaita-for-Steam CSS generation + deployment to Steam/AdwSteam paths + libraryroot rewrite |
| `vesktop` | Vesktop theme CSS generation from Material palette |
| `pear-desktop` | YouTube Music desktop CSS generation + config registration + desktop override injection |
| `sddm` | SDDM `theme.conf` color sync + wallpaper background copy |

### Partially Implemented / Parity Gaps

| Target | Description | Issue |
|---|---|---|
| `gtk-kde` | GTK3/4 CSS, kdeglobals, Darkly.colors, qt5ct/qt6ct generation | Works, but still needs stricter parity validation against upstream iNiR outputs |
| `terminals` | Multi-terminal config generation | Coverage is strong but not yet full parity with upstream Python generator's complete terminal matrix |
| `editors` | VS Code/Zed/OpenCode generation | Functional now, but does not yet fully replicate upstream full theme-generator richness |

### Not Yet Implemented (or only partially ported from shell scripts)

| Area | Remaining Work |
|---|---|
| Chrome policy setup | Automatic privileged policy directory setup is intentionally not handled; create writable policy dirs manually as needed |
| Steam/Pear runtime behavior | CDP live-injection parity from shell scripts is not fully ported |
| Vesktop/Discord | system24-style full parity behavior still incomplete |
| Generator fidelity | Some script-level edge cases and output normalization still need parity passes |

## TODO

### High Priority

- [ ] **GTK/KDE parity hardening** — add output-level regression tests and verify generated GTK/KDE artifacts against upstream iNiR references
- [ ] **Terminal matrix parity** — extend terminal writers to match the full upstream terminal coverage and edge-case formatting
- [ ] **Editor generator fidelity** — deepen VS Code/Zed/OpenCode generation to match upstream theme generator richness

### Medium Priority

- [ ] **Steam live runtime parity** — add missing CDP/live-application behavior from legacy scripts
- [ ] **Pear Desktop live runtime parity** — add missing CDP/live-application behavior from legacy scripts
- [ ] **Vesktop/Discord parity expansion** — complete system24-style behavior and compatibility handling
- [ ] **Target-level golden tests** — add snapshot/golden tests for generated CSS/JSON outputs per target

### Low Priority

- [ ] **Template rendering** — test and validate `{{colors.TOKEN.MODE.hex}}` template system against iNiR templates
- [ ] **Soften colors** — implement HSL saturation reduction for preset themes (currently only in QML)
- [ ] **Image format support** — add WebP and AVIF decoding
- [ ] **Color science/unit test expansion** — broaden tests for scheme generation, harmonization, and fallback handling
- [ ] **Man page / shell completions** — generate from cobra

## Credits

Extracted from [iNiR](https://github.com/snowarch/iNiR) — a Hyprland-based desktop environment using Quickshell.

Color science based on Google's [Material Color Utilities](https://github.com/material-foundation/material-color-utilities) (HCT color space, QuantizeCelebi).

## License

Same as iNiR.
