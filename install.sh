#!/usr/bin/env bash
# =============================================================================
# install.sh — Elegant installer for inir-cli
#
# Builds from source and installs the inir-cli binary to your system.
# Idempotent: safe to re-run at any time.
#
# After installation, prompts whether to create an `inir` symlink.
# Default is NO — both commands coexist. Use --replace-inir to skip the prompt.
#
# Usage:
#   ./install.sh                        # Install to ~/.local/bin, prompt for inir alias
#   ./install.sh --replace-inir         # Create inir symlink (no prompt)
#   ./install.sh --no-replace-inir      # Skip prompt, keep upstream inir (no prompt)
#   ./install.sh --prefix /usr          # Install to /usr/bin (requires sudo)
#   ./install.sh --dry-run              # Show what would happen
#   ./install.sh --uninstall            # Remove inir-cli (and inir symlink if present)
#   ./install.sh --verbose              # Verbose output
#
# Environment variables:
#   GOBIN     — Override binary install directory (takes priority over --prefix)
#   GO_FLAGS  — Additional flags passed to `go build`
#   LD_FLAGS  — Additional linker flags (appended to defaults)
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
BINARY_NAME="inir-cli"
ALIAS_NAME="inir"
MODULE="github.com/yukazakiri/inir-cli"
MIN_GO_VERSION="1.24"
DEFAULT_PREFIX="$HOME/.local"

# ---------------------------------------------------------------------------
# Color helpers (disabled when piped or NO_COLOR is set)
# ---------------------------------------------------------------------------
# Using $'...' ANSI-C quoting so bash converts \033 to real ESC characters.
# This ensures printf/echo render colors correctly regardless of implementation.
if [[ -t 1 ]] && [[ -z "${NO_COLOR:-}" ]]; then
    C_RED=$'\033[0;31m'
    C_GREEN=$'\033[0;32m'
    C_YELLOW=$'\033[0;33m'
    C_BLUE=$'\033[0;34m'
    C_BOLD=$'\033[1m'
    C_DIM=$'\033[2m'
    C_RESET=$'\033[0m'
else
    C_RED='' C_GREEN='' C_YELLOW='' C_BLUE='' C_BOLD='' C_DIM='' C_RESET=''
fi

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
VERBOSE=0

log()       { printf "${C_BOLD}>>>${C_RESET} %s\n" "$*"; }
log_ok()    { printf "${C_GREEN} ✓${C_RESET}  %s\n" "$*"; }
log_warn()  { printf "${C_YELLOW} ⚠${C_RESET}  %s\n" "$*"; }
log_err()   { printf "${C_RED} ✗${C_RESET}  %s\n" "$*" >&2; }
log_step()  { printf "${C_BLUE}  →${C_RESET}  %s\n" "$*"; }
log_dim()   { printf "${C_DIM}    %s${C_RESET}\n" "$*"; }

log_verbose() {
    if [[ "$VERBOSE" -eq 1 ]]; then
        printf "${C_DIM}    %s${C_RESET}\n" "$*"
    fi
}

die() {
    log_err "$@"
    exit 1
}

# ---------------------------------------------------------------------------
# Interactive prompt (yes/no, default=no)
# ---------------------------------------------------------------------------
prompt_yes_no() {
    local question="$1"

    if [[ "$DRY_RUN" -eq 1 ]]; then
        # In dry-run, assume no (the safe default)
        return 1
    fi

    # Skip prompt if not a terminal (piped/CI)
    if [[ ! -t 0 ]]; then
        return 1
    fi

    while true; do
        printf "${C_BOLD}  ?${C_RESET}  %s [y/N] " "$question"
        local answer
        read -r answer
        answer="${answer,,}"  # lowercase

        if [[ -z "$answer" ]]; then
            # Enter pressed — default is No
            return 1
        fi

        case "$answer" in
            y|yes) return 0 ;;
            n|no)  return 1 ;;
            *)     echo "  Please answer y(es) or n(o)." ;;
        esac
    done
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
PREFIX=""
DRY_RUN=0
UNINSTALL=0
REPLACE_INIR=""  # "" = prompt, "yes" = always, "no" = never

usage() {
    cat <<EOF
${C_BOLD}install.sh${C_RESET} — Build and install ${BINARY_NAME}

${C_BOLD}Usage:${C_RESET}
  ./install.sh [OPTIONS]

${C_BOLD}Options:${C_RESET}
  --prefix DIR          Install to DIR/bin (default: ${DEFAULT_PREFIX})
  --replace-inir        Create an ${C_BOLD}inir${C_RESET} symlink replacing the upstream command
  --no-replace-inir     Skip the inir prompt, keep upstream (no prompt)
  --uninstall           Remove ${BINARY_NAME} (and inir symlink if present)
  --dry-run             Show what would happen without doing it
  --verbose, -v        Verbose output
  --help, -h            Show this help message

${C_BOLD}Environment:${C_RESET}
  GOBIN             Override binary install directory
  GO_FLAGS          Additional flags for go build
  LD_FLAGS          Additional linker flags

${C_BOLD}About the inir alias:${C_RESET}
  The upstream iNiR project provides a shell command called ${C_BOLD}inir${C_RESET}.
  After installing, you'll be asked if you want to create an ${C_BOLD}inir${C_RESET}
  symlink pointing to ${BINARY_NAME}. This lets you use the shorter
  ${C_BOLD}inir${C_RESET} command instead of ${C_BOLD}inir-cli${C_RESET}.

  Default answer is ${C_BOLD}No${C_RESET} — both commands coexist:
    ${C_BOLD}inir${C_RESET}     → upstream shell script
    ${C_BOLD}inir-cli${C_RESET} → Go CLI

  If you choose ${C_BOLD}Yes${C_RESET}: the symlink replaces the upstream command.
  The upstream file is backed up as ${C_BOLD}inir.upstream.bak${C_RESET} and
  restored on uninstall.

${C_BOLD}Examples:${C_RESET}
  ./install.sh                          # Install, prompt for inir alias
  ./install.sh --replace-inir           # Install + create inir symlink (no prompt)
  ./install.sh --no-replace-inir        # Install + skip prompt (keep upstream)
  ./install.sh --prefix /usr/local      # Install to /usr/local/bin
  ./install.sh --uninstall              # Remove inir-cli and inir symlink
  ./install.sh --dry-run --verbose      # Preview with full output
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --prefix)
            shift
            PREFIX="${1:?--prefix requires a directory argument}"
            ;;
        --replace-inir)     REPLACE_INIR="yes" ;;
        --no-replace-inir)  REPLACE_INIR="no" ;;
        --uninstall)         UNINSTALL=1 ;;
        --dry-run)           DRY_RUN=1 ;;
        --verbose|-v)        VERBOSE=1 ;;
        --help|-h)           usage ;;
        *)                   die "Unknown option: $1 (try --help)" ;;
    esac
    shift
done

# ---------------------------------------------------------------------------
# Resolve install directory
# ---------------------------------------------------------------------------
resolve_bindir() {
    if [[ -n "${GOBIN:-}" ]]; then
        echo "$GOBIN"
        return
    fi
    local prefix="${PREFIX:-$DEFAULT_PREFIX}"
    echo "${prefix}/bin"
}

# ---------------------------------------------------------------------------
# Utility: compare semver versions (returns 0 if $1 >= $2)
# ---------------------------------------------------------------------------
version_gte() {
    local v1="$1" v2="$2"
    # Strip any "go" prefix (e.g., go1.24.0 → 1.24.0)
    v1="${v1#go}"
    v2="${v2#go}"

    local IFS=.
    read -ra v1_parts <<< "$v1"
    read -ra v2_parts <<< "$v2"

    for i in 0 1 2; do
        local a="${v1_parts[$i]:-0}"
        local b="${v2_parts[$i]:-0}"
        if (( a > b )); then return 0; fi
        if (( a < b )); then return 1; fi
    done
    return 0
}

# ---------------------------------------------------------------------------
# Detect Go
# ---------------------------------------------------------------------------
detect_go() {
    if ! command -v go &>/dev/null; then
        die "Go is not installed. Install Go ${MIN_GO_VERSION}+ from https://go.dev/dl/"
    fi

    local go_version
    go_version="$(go version | awk '{print $3}')"
    go_version="${go_version#go}"

    log_verbose "Detected Go: ${go_version}"

    if ! version_gte "$go_version" "$MIN_GO_VERSION"; then
        die "Go ${MIN_GO_VERSION}+ required, found ${go_version}. Upgrade at https://go.dev/dl/"
    fi

    log_ok "Go ${go_version} detected"
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build_binary() {
    local bindir="$1"
    local binary="${bindir}/${BINARY_NAME}"

    # Resolve git info for ldflags (best-effort)
    local commit="unknown" version="dev" buildtime
    buildtime="$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown")"

    if git rev-parse --short HEAD &>/dev/null; then
        commit="$(git rev-parse --short HEAD)"
        # Use git tag if available
        local tag
        tag="$(git describe --tags --exact-match 2>/dev/null || true)"
        if [[ -n "$tag" ]]; then
            version="${tag#v}"
        fi
    fi

    local ldflags="-s -w"
    ldflags+=" -X main.version=${version}"
    ldflags+=" -X main.commit=${commit}"
    ldflags+=" -X main.buildTime=${buildtime}"
    ldflags+=" ${LD_FLAGS:-}"

    local go_flags="-trimpath ${GO_FLAGS:-}"

    log "Building ${C_BOLD}${BINARY_NAME}${C_RESET} (version=${version}, commit=${commit})"

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log_dim "Would run: go build ${go_flags} -ldflags '${ldflags}' -o ${binary} ."
        return
    fi

    log_step "Compiling..."
    if ! go build ${go_flags} -ldflags "${ldflags}" -o "${binary}" .; then
        die "Build failed. Check errors above."
    fi

    log_ok "Built ${binary}"
    log_verbose "  ldflags: ${ldflags}"
    log_verbose "  flags:   ${go_flags}"
}

# ---------------------------------------------------------------------------
# Install: ensure directory exists and binary is in place
# ---------------------------------------------------------------------------
install_binary() {
    local bindir="$1"

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log_dim "Would ensure directory: ${bindir}"
        log_dim "Would install: ${bindir}/${BINARY_NAME}"
        return
    fi

    # Create bindir if needed
    if [[ ! -d "$bindir" ]]; then
        log_step "Creating directory: ${bindir}"
        mkdir -p "$bindir"
    fi

    # Make binary executable (idempotent)
    chmod +x "${bindir}/${BINARY_NAME}" 2>/dev/null || true

    log_ok "Installed to ${bindir}/${BINARY_NAME}"
}

# ---------------------------------------------------------------------------
# inir alias: create or remove symlink
# ---------------------------------------------------------------------------
setup_inir_alias() {
    local bindir="$1"
    local target="${bindir}/${BINARY_NAME}"
    local alias_path="${bindir}/${ALIAS_NAME}"

    # Determine whether to create the alias
    local create_alias=""

    if [[ "$REPLACE_INIR" == "yes" ]]; then
        create_alias="yes"
    elif [[ "$REPLACE_INIR" == "no" ]]; then
        create_alias="no"
    else
        # Interactive prompt (default: No)
        echo ""
        log "The upstream iNiR project provides a shell command called ${C_BOLD}inir${C_RESET}."
        log "You can create an ${C_BOLD}inir${C_RESET} symlink that points to ${C_BOLD}${BINARY_NAME}${C_RESET}."
        echo ""
        log_dim "  Yes → typing ${C_BOLD}inir${C_RESET} will run ${C_BOLD}${BINARY_NAME}${C_RESET} (replaces upstream)"
        log_dim "  No  → ${C_BOLD}inir${C_RESET} stays as the upstream shell script, use ${C_BOLD}${BINARY_NAME}${C_RESET} for the Go CLI"
        echo ""

        if prompt_yes_no "Create an 'inir' symlink pointing to inir-cli?"; then
            create_alias="yes"
        else
            create_alias="no"
        fi
    fi

    if [[ "$create_alias" == "no" ]]; then
        echo ""
        log_ok "Keeping upstream ${C_BOLD}inir${C_RESET} command as-is"
        log_dim "Use ${C_BOLD}${BINARY_NAME}${C_RESET} to run the Go CLI"
        log_dim "Re-run with ${C_BOLD}--replace-inir${C_RESET} to create an ${C_BOLD}inir${C_RESET} symlink later"
        return
    fi

    # --- create_alias == "yes" ---

    # Check if alias already points to our binary
    if [[ -L "$alias_path" ]]; then
        local current_target
        current_target="$(readlink -f "$alias_path" 2>/dev/null || true)"
        if [[ "$current_target" == "$target" ]] || [[ "$(basename "$current_target")" == "${BINARY_NAME}" ]]; then
            log_ok "Symlink ${C_BOLD}${ALIAS_NAME}${C_RESET} → ${BINARY_NAME} already exists"
            return
        fi
        # Symlink exists but points elsewhere — update it
        if [[ "$DRY_RUN" -eq 1 ]]; then
            log_dim "Would update symlink: ${alias_path} → ${target}"
            return
        fi
        log_step "Updating symlink: ${alias_path} → ${target}"
        ln -sf "$target" "$alias_path"
        log_ok "Symlink ${C_BOLD}${ALIAS_NAME}${C_RESET} → ${BINARY_NAME} updated"
        return
    fi

    # Check if a real file (not symlink) exists at alias path
    if [[ -e "$alias_path" ]] && [[ ! -L "$alias_path" ]]; then
        local backup="${alias_path}.upstream.bak"
        echo ""
        log_warn "An existing file exists at ${C_BOLD}${alias_path}${C_RESET}"
        log_dim "This appears to be the upstream iNiR shell command."
        log_dim "It will be backed up as ${C_BOLD}inir.upstream.bak${C_RESET} and restored on uninstall."
        echo ""

        if [[ "$DRY_RUN" -eq 1 ]]; then
            log_dim "Would back up: ${alias_path} → ${backup}"
            log_dim "Would create symlink: ${alias_path} → ${target}"
            return
        fi

        log_step "Backing up upstream: ${alias_path} → ${backup}"
        mv "$alias_path" "$backup"
        log_dim "Original saved as ${backup}"
        log_step "Creating symlink: ${alias_path} → ${target}"
        ln -s "$target" "$alias_path"
        log_ok "Symlink ${C_BOLD}${ALIAS_NAME}${C_RESET} → ${BINARY_NAME} created (upstream backed up)"
        return
    fi

    # No existing file or symlink — create fresh
    if [[ "$DRY_RUN" -eq 1 ]]; then
        log_dim "Would create symlink: ${alias_path} → ${target}"
        return
    fi

    log_step "Creating symlink: ${alias_path} → ${target}"
    ln -s "$target" "$alias_path"
    log_ok "Symlink ${C_BOLD}${ALIAS_NAME}${C_RESET} → ${BINARY_NAME} created"
    log_dim "You can now use ${C_BOLD}inir${C_RESET} as a shorthand for ${C_BOLD}${BINARY_NAME}${C_RESET}"
}

# ---------------------------------------------------------------------------
# PATH: ensure install directory is on PATH
# ---------------------------------------------------------------------------
ensure_path() {
    local bindir="$1"

    # Check if already on PATH
    if echo ":$PATH:" | grep -q ":${bindir}:" ; then
        log_ok "${bindir} is on PATH"
        return
    fi

    log_warn "${bindir} is not on PATH"

    # Find the appropriate shell rc file
    local rc_file=""
    local shell_name="$(basename "${SHELL:-bash}")"
    case "$shell_name" in
        zsh)  rc_file="$HOME/.zshrc" ;;
        bash) rc_file="$HOME/.bashrc" ;;
        fish) rc_file="$HOME/.config/fish/config.fish" ;;
        *)    rc_file="" ;;
    esac

    if [[ -z "$rc_file" ]]; then
        log_dim "Add ${bindir} to your PATH manually."
        return
    fi

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log_dim "Would add PATH export to ${rc_file}"
        return
    fi

    # Check if the rc file already has this bindir in PATH (from a previous install)
    if [[ -f "$rc_file" ]] && grep -q "${bindir}" "$rc_file" 2>/dev/null; then
        log_dim "PATH entry for ${bindir} already exists in ${rc_file}"
        log_dim "Run: source ${rc_file}  (or restart your shell)"
        return
    fi

    # Append PATH export
    local export_line=""
    case "$shell_name" in
        fish) export_line="set -gx PATH ${bindir} \$PATH" ;;
        *)    export_line="export PATH=\"${bindir}:\$PATH\"" ;;
    esac

    log_step "Adding ${bindir} to PATH in ${rc_file}"
    {
        echo ""
        echo "# Added by inir-cli installer"
        echo "${export_line}"
    } >> "$rc_file"

    log_ok "PATH configured in ${rc_file}"
    log_dim "Run: source ${rc_file}  (or restart your shell)"
}

# ---------------------------------------------------------------------------
# Verify installation
# ---------------------------------------------------------------------------
verify_install() {
    local bindir="$1"
    local binary="${bindir}/${BINARY_NAME}"

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log_dim "Would verify: ${binary} --help"
        return
    fi

    if [[ ! -x "$binary" ]]; then
        die "Binary not found at ${binary}"
    fi

    log_step "Verifying installation..."
    if "$binary" --help &>/dev/null; then
        log_ok "Verification passed: ${BINARY_NAME} is working"
    else
        log_warn "Binary exists but --help returned an error"
    fi

    # Show version info if available
    local version_output
    version_output="$("$binary" --version 2>/dev/null || echo "unknown")"
    log_dim "Version: ${version_output}"

    # Check inir alias if it exists
    local alias_path="${bindir}/${ALIAS_NAME}"
    if [[ -L "$alias_path" ]]; then
        local alias_target
        alias_target="$(readlink "$alias_path")"
        log_dim "Alias: ${ALIAS_NAME} → ${alias_target}"
    fi
}

# ---------------------------------------------------------------------------
# Systemd user service (auto-start on login)
# ---------------------------------------------------------------------------
setup_systemd_service() {
    local bindir="$1"
    local binary="${bindir}/${BINARY_NAME}"

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log_dim "Would attempt: ${binary} service install"
        log_dim "Would attempt: ${binary} service enable"
        return
    fi

    if ! command -v systemctl &>/dev/null; then
        log_dim "systemctl not found; skipping iNiR user service setup"
        return
    fi

    # Best-effort: only proceed when systemd user manager is reachable.
    if ! systemctl --user show-environment >/dev/null 2>&1; then
        log_dim "systemd --user not reachable in this session; skipping service setup"
        return
    fi

    log_step "Installing iNiR user service"
    if "$binary" service install >/dev/null 2>&1; then
        log_ok "Installed inir.service"
    else
        log_warn "Failed to install inir.service (non-fatal)"
        return
    fi

    log_step "Enabling iNiR user service for compositor startup"
    if "$binary" service enable >/dev/null 2>&1; then
        log_ok "Enabled inir.service"
    else
        log_warn "Could not enable inir.service (no supported compositor detected?)"
        log_dim "You can run manually later: ${BINARY_NAME} service enable"
    fi
}

# ---------------------------------------------------------------------------
# Uninstall
# ---------------------------------------------------------------------------
do_uninstall() {
    local bindir="$1"
    local binary="${bindir}/${BINARY_NAME}"
    local alias_path="${bindir}/${ALIAS_NAME}"

    # Remove the main binary
    if [[ -f "$binary" ]]; then
        if [[ "$DRY_RUN" -eq 1 ]]; then
            log_dim "Would remove: ${binary}"
        else
            rm -f "$binary"
            log_ok "Removed ${binary}"
        fi
    else
        log_warn "${binary} not found — already removed"
    fi

    # Remove the inir symlink if it points to inir-cli
    if [[ -L "$alias_path" ]]; then
        local alias_target
        alias_target="$(readlink -f "$alias_path" 2>/dev/null || true)"
        if [[ "$(basename "$alias_target")" == "${BINARY_NAME}" ]]; then
            if [[ "$DRY_RUN" -eq 1 ]]; then
                log_dim "Would remove symlink: ${alias_path}"
            else
                rm -f "$alias_path"
                log_ok "Removed ${ALIAS_NAME} symlink"
            fi
        else
            log_dim "Keeping ${ALIAS_NAME} — it points to something else (${alias_target})"
        fi
    elif [[ -e "$alias_path" ]]; then
        log_dim "Keeping ${ALIAS_NAME} — it's not a symlink (likely the upstream command)"
    fi

    # Check for backed-up upstream
    local backup="${alias_path}.upstream.bak"
    if [[ -f "$backup" ]]; then
        if [[ "$DRY_RUN" -eq 1 ]]; then
            log_dim "Would restore upstream: ${backup} → ${alias_path}"
        else
            log_step "Restoring upstream inir from backup"
            mv "$backup" "$alias_path"
            log_ok "Restored upstream ${ALIAS_NAME} from backup"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    local bindir
    bindir="$(resolve_bindir)"

    echo ""
    log "${C_BOLD}${BINARY_NAME}${C_RESET} installer"
    log_dim "Install directory: ${bindir}"
    echo ""

    if [[ "$UNINSTALL" -eq 1 ]]; then
        do_uninstall "$bindir"
        echo ""
        log "Uninstall complete"
        return 0
    fi

    # Step 1: Detect Go
    detect_go

    # Step 2: Build
    build_binary "$bindir"

    # Step 3: Install
    install_binary "$bindir"

    # Step 4: inir alias
    setup_inir_alias "$bindir"

    # Step 5: PATH
    ensure_path "$bindir"

    # Step 6: Verify
    verify_install "$bindir"

    # Step 7: User service wiring (best-effort)
    setup_systemd_service "$bindir"

    echo ""
    log "Installation complete! ${C_GREEN}${C_BOLD}${BINARY_NAME}${C_RESET} is ready."
    echo ""

    # Hint for first-time users
    if [[ "$DRY_RUN" -eq 0 ]]; then
        local alias_path="${bindir}/${ALIAS_NAME}"
        if [[ -L "$alias_path" ]]; then
            log_dim "Quick start:"
            log_dim "  inir generate --image /path/to/wallpaper.jpg"
            log_dim "  inir scheme --list"
            log_dim "  inir theme apply all"
            echo ""
            log_dim "(You can also use ${C_BOLD}${BINARY_NAME}${C_RESET} instead of ${C_BOLD}inir${C_RESET})"
        else
            log_dim "Quick start:"
            log_dim "  inir-cli generate --image /path/to/wallpaper.jpg"
            log_dim "  inir-cli scheme --list"
            log_dim "  inir-cli theme apply all"
        fi
    fi
}

main
