# External Theme Target Spec

`inir-cli` supports community-extensible target application via auto-discovered JSON spec files.

Add one file and `inir-cli theme apply ...` can run it.

## Discovery Paths

Targets are discovered from these directories (in order):

1. `${INIR_THEME_TARGETS_DIR}` (colon-separated)
2. `$(dirname <config.json>)/targets/`
3. `~/.config/inir/targets/`
4. `~/.config/inir-cli/targets/`

Use `inir-cli theme list-targets` to verify discovery.

## Minimal Example

```json
{
  "id": "my-app",
  "type": "command",
  "description": "Apply generated palette to My App",
  "command": "/usr/local/bin/my-app-theme-apply"
}
```

## Full Example

```json
{
  "id": "my-app",
  "label": "My App",
  "description": "Apply generated palette to My App",
  "type": "command",
  "command": "/usr/local/bin/my-app-theme-apply",
  "args": ["--mode", "material"],
  "inputs": ["palette.json", "terminal.json"],
  "env": {
    "MY_APP_THEME_MODE": "material"
  },
  "enabled": true
}
```

## Fields

- `id` (required): target id. Pattern: `^[a-z0-9][a-z0-9-]*$`
- `type` (optional): currently must be `"command"` (defaults to `"command"`)
- `command` (required): executable path/name
- `args` (optional): string array of arguments
- `description` (optional): shown in `theme list-targets`
- `label` (optional): display name for future UI/UX
- `inputs` (optional): declared contract dependencies (documentation/validation aid)
- `env` (optional): key/value env overrides for target execution
- `enabled` (optional): set `false` to disable without deleting file

## Runtime Environment Exposed to External Targets

`inir-cli` injects these environment variables when running external targets:

- `INIR_OUTPUT_DIR`
- `INIR_COLORS_JSON`
- `INIR_PALETTE_JSON`
- `INIR_TERMINAL_JSON`
- `INIR_THEME_META_JSON`
- `INIR_MATERIAL_SCSS`
- `INIR_CONFIG_JSON`

## Scaffold Command

Generate a starter spec with:

```bash
inir-cli theme scaffold-target my-app \
  --command /usr/local/bin/my-app-theme-apply \
  --description "Apply generated palette to My App"
```

Useful options:

- `--arg` (repeatable)
- `--input` (repeatable)
- `--env KEY=VALUE` (repeatable)
- `--enabled=false`
- `--dir <path>` (override destination)
- `--force` (overwrite existing file)

## Error Handling

- Invalid JSON/spec shape → apply fails with clear parse/validation error
- Duplicate IDs across discovery paths → apply fails fast
- Command execution failures → surfaced in CLI error + desktop notification (if `notify-send` is available)
