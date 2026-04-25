#!/usr/bin/env bash
# =============================================================================
# install.sh — Elegant installer for inir-cli
#
# Builds from source and installs the inir-cli binary to your system.
# Idempotent: safe to re-run at any time.
#
# By default, this does NOT replace the upstream `inir` command.
# Use --replace-inir to create an `inir` symlink that points to inir-cli.
#
# Usage:
#   ./install.sh                        # Install to ~/.local/bin (does NOT replace inir)
#   ./install.sh --replace-inir         # Install + create inir symlink (replaces upstream)
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
if [[ -t 1 ]] && [[ -z "${NO_COLOR:-}" ]]; then
    C_RED='\033[0;31m'
    C_GREEN='\033[0;32m'
    C_YELLOW='\033[0;33m'
    C_BLUE='\033[0;34m'
    C_BOLD='\033[1m'
    C_DIM='\033[2m'
    C_RESET='\033[0m'
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
# Argument parsing
# ---------------------------------------------------------------------------
PREFIX=""
DRY_RUN=0
UNINSTALL=0
REPLACE_INIR="no"  # "yes" = create inir symlink, "no" = keep upstream (default)

usage() {
    cat <<EOF
${C_BOLD}install.sh${C_RESET} — Build and install ${BINARY_NAME}

${C_BOLD}Usage:${C_RESET}
  ./install.sh [OPTIONS]

${C_BOLD}Options:${C_RESET}
  --prefix DIR          Install to DIR/bin (default: ${DEFAULT_PREFIX})
  --replace-inir        Create an ${C_BOLD}inir${C_RESET} symlink replacing the upstream command
  --uninstall           Remove ${BINARY_NAME} (and inir symlink if present)
  --dry-run             Show what would happen without doing it
  --verbose, -v        Verbose output
  --help, -h            Show this help message

${C_BOLD}Environment:${C_RESET}
  GOBIN             Override binary install directory
  GO_FLAGS          Additional flags for go build
  LD_FLAGS          Additional linker flags

${C_BOLD}About the inir alias:${C_RESET}
  By default, this installer does ${C_BOLD}not${C_RESET} replace the upstream ${C_BOLD}inir${C_RESET} command.
  Both commands coexist: ${C_BOLD}inir${C_RESET} runs the upstream shell script,
  ${C_BOLD}inir-cli${C_RESET} runs the Go CLI.

  Pass ${C_BOLD}--replace-inir${C_RESET} to create an ${C_BOLD}inir${C_RESET} symlink pointing
  to ${BINARY_NAME}. This replaces the upstream command — typing ${C_BOLD}inir${C_RESET}
  will then run the Go CLI instead. The upstream file is backed up as
  ${C_BOLD}inir.upstream.bak${C_RESET} and restored on uninstall.

${C_BOLD}Examples:${C_RESET}
  ./install.sh                          # Install to ~/.local/bin (inir stays as upstream)
  ./install.sh --replace-inir           # Install + create inir symlink (replaces upstream)
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

    # Default: do not replace upstream inir
    if [[ "$REPLACE_INIR" != "yes" ]]; then
        # Check if an inir symlink already points to us (from a previous --replace-inir install)
        if [[ -L "$alias_path" ]]; then
            local current_target
            current_target="$(readlink -f "$alias_path" 2>/dev/null || true)"
            if [[ "$current_target" == "$target" ]] || [[ "$(basename "$current_target")" == "${BINARY_NAME}" ]]; then
                log_ok "Symlink ${C_BOLD}${ALIAS_NAME}${C_RESET} → ${BINARY_NAME} already exists"
                return
            fi
        fi

        echo ""
        log_ok "Keeping upstream ${C_BOLD}inir${C_RESET} command as-is"
        log_dim "Use ${C_BOLD}${BINARY_NAME}${C_RESET} to run the Go CLI"
        log_dim "Re-run with ${C_BOLD}--replace-inir${C_RESET} to create an ${C_BOLD}inir${C_RESET} symlink"
        return
    fi

    # --- REPLACE_INIR == "yes" ---

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

    # Check if a real file (not symlink) exists at alias path — back it up
    if [[ -e "$alias_path" ]] && [[ ! -L "$alias_path" ]]; then
        local backup="${alias_path}.upstream.bak"
        if [[ "$DRY_RUN" -eq 1 ]]; then
            log_dim "Would back up: ${alias_path} → ${backup}"
            log_dim "Would create symlink: ${alias_path} → ${target}"
            return
        fi
        log_warn "An existing file exists at ${C_BOLD}${alias_path}${C_RESET}"
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