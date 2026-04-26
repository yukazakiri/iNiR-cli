# cmd/ — Developer Guide

This directory contains all CLI command wiring and orchestration logic for `inir-cli`. It uses [Cobra](https://github.com/spf13/cobra) for command parsing and delegates all heavy computation to packages under `internal/`.

## Architecture Overview

```
User input → cmd/ (cobra wiring + orchestration) → internal/ (business logic)
```

The `cmd/` package is responsible for:
- Defining the cobra command tree (flags, args, help text)
- Loading configuration and resolving paths (XDG, config files)
- Orchestrating the pipeline: generate → write contract → apply targets
- IPC command routing and validation
- External target discovery and execution

It does **not** contain color science, preset data, or target implementation logic — those live in `internal/`.

## File Organization

Files are grouped by domain. Each group has a clear responsibility boundary.

### Root & Command Wiring

| File | Purpose |
|---|---|
| `root.go` | Root cobra command, `generate`, `apply`, `theme`, `auto-detect` top-level commands. Entry point via `Execute()`. |
| `theme.go` | `theme generate` and `theme apply` subcommands. Flag definitions (`--image`, `--color`, `--mode`, etc.), config loading, XDG resolution, and the `runGenerate`/`runThemeApply` orchestration functions. |
| `scheme.go` | `scheme` command for applying built-in preset themes. Terminal color harmonization, preset resolution, `--list`, `--random`, `--apply` flags. |
| `register.go` | Built-in target registration. Maps target names (e.g. `"gtk-kde"`, `"terminals"`) to their `Applier` factories. |
| `targets.go` | Helper to list all registered built-in target names. |

### Theme Pipeline (generate → contract → apply)

The theme pipeline is the core workflow: generate colors, write them to a contract directory, then apply to targets.

| File | Purpose |
|---|---|
| `theme_pipeline_contract.go` | Defines `outputContract` — the centralized file layout for generated output (`colors.json`, `palette.json`, `terminal.json`, `theme-meta.json`, `material_colors.scss`). All generation paths write through this contract. |
| `theme_pipeline_apply.go` | `applyThemeTargets()` — unified apply orchestration. Resolves requested targets (including `"all"`), runs built-in appliers, runs external targets, collects failures, and surfaces errors. |
| `theme_pipeline_targets.go` | External target discovery and execution. Loads JSON specs from `INIR_THEME_TARGETS_DIR`, config-relative, `~/.config/inir/targets/`, and `~/.config/inir-cli/targets/`. Validates specs, injects environment variables, and executes external commands. |
| `theme_pipeline_notify.go` | Desktop notification integration via `notify-send`. Best-effort — silently skipped if `notify-send` is unavailable. Used to surface apply failures. |
| `theme_targets_cmd.go` | `theme list-targets` and `theme scaffold-target` cobra commands. List shows built-in + discovered external targets. Scaffold creates a JSON spec file for community onboarding. |

### Chromium Contract

| File | Purpose |
|---|---|
| `chromium_contract.go` | Writes `chromium.theme` (RGB seed for browser theming) and `color.txt` (compatibility seed). Derives the seed from `surface_container_low` → `surface` → `background` fallback chain. |

### IPC (iNiR Shell Communication)

The IPC subsystem mirrors the upstream `inir` command for Quickshell IPC targets.

| File | Purpose |
|---|---|
| `ipc.go` | Cobra command wiring for `ipc` (raw passthrough) and per-target subcommands (e.g. `overview toggle`). Routes to `parseIPCPrefixArgs` → `runIPCCommand`. |
| `ipc_registry.go` | Types (`ipcTarget`, `ipcFunction`) and lookup helpers (`findIPCTarget`, `ipcAliasesForTarget`). Assembles the final registry from generated + override data. |
| `ipc_registry_generated.go` | Auto-generated upstream IPC target data (45 targets, kebab-case aliases). **Do not manually edit** — regenerate from upstream. |
| `ipc_registry_overrides.go` | Manual override layer. Add or modify targets here without touching generated code. Merged at init time. |
| `ipc_parse.go` | Argument parsing for IPC commands: `-c`/`--config` prefix extraction, kebab-case normalization, function validation. |
| `ipc_runtime.go` | Runtime directory resolution (`INIR_RUNTIME_DIR`, XDG, system paths), `qs` binary execution, shell payload validation. |
| `ipc_help.go` | Per-target help formatting. Lists available functions, family, and example keybinds. |

### Tests & Benchmarks

| File | Purpose |
|---|---|
| `ipc_test.go` | IPC normalization, validation, parse, runner, raw passthrough, settings default, target command, runtime errors, help. |
| `ipc_benchmark_test.go` | IPC dispatch benchmarks (Go vs upstream shell). |
| `color_pipeline_benchmark_test.go` | Color generate + apply pipeline benchmarks. |
| `chromium_contract_test.go` | Chromium theme contract tests. |
| `scheme_test.go` | Scheme command tests. |
| `theme_pipeline_apply_test.go` | Apply orchestration + external target tests. |
| `theme_pipeline_notify_test.go` | Notification helper tests. |
| `theme_pipeline_targets_test.go` | External target discovery tests. |
| `theme_scaffold_cmd_test.go` | Scaffold command tests. |

## Command Tree

```
inir-cli
├── generate                     # Generate palette from wallpaper/seed color
├── apply [targets...]           # (redirects to theme apply)
├── scheme [theme-name]          # Apply a built-in preset theme
├── auto-detect [image]          # Detect best scheme variant for an image
├── theme
│   ├── generate                 # Generate (namespaced)
│   ├── apply [targets...]       # Apply generated colors to targets
│   ├── list-targets             # List built-in + external targets
│   └── scaffold-target <id>     # Create external target JSON spec
├── ipc <target> <function>       # Raw Quickshell IPC passthrough
├── <target> <function>          # Upstream-style IPC (e.g. overview toggle)
└── ...                          # 45 IPC target subcommands
```

## Data Flow

### Generate Flow
```
User runs: inir-cli generate --image wallpaper.jpg --mode dark
    │
    ├─ loadConfig()          → reads ~/.config/inir/config.json
    ├─ resolveOutputDir()    → XDG_STATE_HOME/quickshell/user/generated
    ├─ newOutputContract()   → sets up file paths (colors.json, palette.json, etc.)
    ├─ color.Generate()      → internal/color: HCT, QuantizeCelebi, scheme, terminal
    ├─ contract.WriteJSON()  → writes colors.json, palette.json, terminal.json, theme-meta.json
    ├─ contract.WriteSCSS()  → writes material_colors.scss
    ├─ writeChromiumThemeContracts() → writes chromium.theme + color.txt
    └─ template.RenderAll()  → (optional) renders user templates
```

### Apply Flow
```
User runs: inir-cli theme apply gtk-kde terminals chrome
    │
    ├─ loadConfig()
    ├─ contract.RequireColors()  → verifies colors.json exists
    ├─ discoverExternalTargets() → loads JSON specs from target dirs
    ├─ resolveRequestedTargets() → expands "all", validates names
    │
    ├─ For each target:
    │   ├─ Built-in? → targetpkg.GetApplier(id).Apply(ctx)
    │   └─ External? → runExternalTarget(spec, contract, configPath)
    │
    ├─ Collect failures
    ├─ notifyApplyFailures()  → desktop notification via notify-send
    └─ Return aggregated error
```

### IPC Flow
```
User runs: inir-cli overview toggle
    │
    ├─ parseIPCPrefixArgs()    → extracts -c/--config if present
    ├─ normalizeIPCTarget()    → kebab-case → camelCase lookup
    ├─ validateIPCFunction()   → checks function exists for target
    ├─ resolveIPCRuntimeDir()  → INIR_RUNTIME_DIR → XDG → system paths
    └─ runQSIPCCommand()       → exec qs -p <dir> ipc call <target> <function> [args]
```

## Naming Conventions

- **`theme_pipeline_*.go`** — Files in the theme pipeline orchestration layer. These coordinate between config, contract, targets, and notifications.
- **`ipc_*.go`** — Files in the IPC subsystem. Split into registry, parsing, runtime, and help.
- **`*_test.go`** — Test files mirror their source file name.
- **`*_benchmark_test.go`** — Benchmark files for performance-critical paths.

## Adding a New Built-in Target

1. Create `internal/target/mytarget/mytarget.go` implementing the `Applier` interface.
2. Register it in `cmd/register.go`:
   ```go
   targetpkg.Register("my-target", func() targetpkg.Applier { return &mytarget.Applier{} })
   ```
3. Add tests in `internal/target/mytarget/mytarget_test.go`.

## Adding a New IPC Target

1. If it's an upstream target, regenerate `ipc_registry_generated.go` from the upstream registry.
2. If it's a custom override, add it to `ipc_registry_overrides.go`.
3. Add kebab-case alias in `ipcAliasOverrides` if needed.

## Adding a New Theme Pipeline Step

1. Create `cmd/theme_pipeline_mystep.go` for orchestration logic.
2. Wire it into `runGenerate()` or `runThemeApply()` in `theme.go`.
3. Add tests in `cmd/theme_pipeline_mystep_test.go`.