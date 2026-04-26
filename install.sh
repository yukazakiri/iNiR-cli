#!/usr/bin/env bash
#
# iNiR CLI Installer
# Builds and installs inir-cli from source.
#
# Usage:
#   ./install.sh
#   ./install.sh --replace-inir
#   ./install.sh --no-replace-inir
#   ./install.sh --prefix /usr/local
#   ./install.sh --uninstall
#

set -euo pipefail

BINARY_NAME="inir-cli"
ALIAS_NAME="inir"
MIN_GO_VERSION="1.24"
DEFAULT_PREFIX="$HOME/.local"

VERBOSE=0
DRY_RUN=0
UNINSTALL=0
PREFIX=""
REPLACE_INIR="" # ""=prompt, yes=always, no=never

if [[ -t 1 ]] && [[ -z "${NO_COLOR:-}" ]]; then
  RED=$'\033[0;31m'
  GREEN=$'\033[0;32m'
  YELLOW=$'\033[1;33m'
  BLUE=$'\033[0;34m'
  CYAN=$'\033[0;36m'
  BOLD=$'\033[1m'
  DIM=$'\033[2m'
  NC=$'\033[0m'
else
  RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' DIM='' NC=''
fi

print_header() {
  printf "${CYAN}"
  cat <<'EOF'
+------------------------------------------------------------------+
|                                                                  |
|      _       _      _         _ _                                |
|     (_)     (_)    | |       | (_)                               |
|      _ _ __  _ _ __| |   ____| |_                                |
|     | | '_ \| | '__| |  / __/| | |                               |
|     | | | | | | |  | | | (__ | | |                               |
|     |_|_| |_|_|_|  |_|  \___||_|_|                               |
|                                                                  |
|                 iNiR Desktop Theme CLI Installer                 |
|                                                                  |
+------------------------------------------------------------------+
EOF
  printf "${NC}\n"
}

log_info()    { printf "${BLUE}[INFO]${NC} %s\n" "$*"; }
log_success() { printf "${GREEN}[ OK ]${NC} %s\n" "$*"; }
log_warn()    { printf "${YELLOW}[WARN]${NC} %s\n" "$*"; }
log_error()   { printf "${RED}[ERR ]${NC} %s\n" "$*" >&2; }
log_step()    { printf "${CYAN}[....]${NC} %s\n" "$*"; }
log_verbose() {
  if [[ "$VERBOSE" -eq 1 ]]; then
    printf "${DIM}%s${NC}\n" "$*"
  fi
  return 0
}

die() {
  log_error "$*"
  exit 1
}

usage() {
  cat <<EOF
${BOLD}install.sh${NC} - Build and install ${BINARY_NAME}

${BOLD}Usage:${NC}
  ./install.sh [OPTIONS]

${BOLD}Options:${NC}
  --prefix DIR          Install to DIR/bin (default: ${DEFAULT_PREFIX})
  --replace-inir        Create an ${ALIAS_NAME} symlink to ${BINARY_NAME}
  --no-replace-inir     Skip symlink prompt and keep upstream ${ALIAS_NAME}
  --uninstall           Remove ${BINARY_NAME} and managed alias/backup
  --dry-run             Show what would happen without changes
  --verbose, -v         Enable verbose logs
  --help, -h            Show this message

${BOLD}Environment:${NC}
  GOBIN                 Override binary install directory
  GO_FLAGS              Extra flags for go build
  LD_FLAGS              Extra linker flags

${BOLD}Examples:${NC}
  ./install.sh
  ./install.sh --replace-inir
  ./install.sh --prefix /usr/local
  ./install.sh --dry-run --verbose
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)
      shift
      PREFIX="${1:?--prefix requires a directory argument}"
      ;;
    --replace-inir) REPLACE_INIR="yes" ;;
    --no-replace-inir) REPLACE_INIR="no" ;;
    --uninstall) UNINSTALL=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --verbose|-v) VERBOSE=1 ;;
    --help|-h) usage ;;
    *) die "Unknown option: $1 (try --help)" ;;
  esac
  shift
done

resolve_bindir() {
  if [[ -n "${GOBIN:-}" ]]; then
    printf "%s\n" "$GOBIN"
    return
  fi
  local prefix="${PREFIX:-$DEFAULT_PREFIX}"
  printf "%s/bin\n" "$prefix"
}

version_gte() {
  local v1="${1#go}" v2="${2#go}"
  local IFS=.
  read -ra p1 <<< "$v1"
  read -ra p2 <<< "$v2"

  for i in 0 1 2; do
    local a="${p1[$i]:-0}"
    local b="${p2[$i]:-0}"
    if (( a > b )); then return 0; fi
    if (( a < b )); then return 1; fi
  done
  return 0
}

detect_go() {
  if ! command -v go >/dev/null 2>&1; then
    die "Go is not installed. Install Go ${MIN_GO_VERSION}+ from https://go.dev/dl/"
  fi

  local go_version
  go_version="$(go version | awk '{print $3}')"
  go_version="${go_version#go}"
  log_verbose "Detected Go version: ${go_version}"

  if ! version_gte "$go_version" "$MIN_GO_VERSION"; then
    die "Go ${MIN_GO_VERSION}+ required, found ${go_version}"
  fi

  log_success "Go ${go_version} detected"
}

build_binary() {
  local bindir="$1"
  local binary="${bindir}/${BINARY_NAME}"

  local commit="unknown"
  local version="dev"
  local build_time
  build_time="$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || printf 'unknown')"

  if git rev-parse --short HEAD >/dev/null 2>&1; then
    commit="$(git rev-parse --short HEAD)"
    local tag
    tag="$(git describe --tags --exact-match 2>/dev/null || true)"
    if [[ -n "$tag" ]]; then
      version="${tag#v}"
    fi
  fi

  local ldflags="-s -w"
  ldflags+=" -X main.version=${version}"
  ldflags+=" -X main.commit=${commit}"
  ldflags+=" -X main.buildTime=${build_time}"
  ldflags+=" ${LD_FLAGS:-}"

  local go_flags="-trimpath ${GO_FLAGS:-}"

  log_info "Building ${BINARY_NAME} (version=${version}, commit=${commit})"

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log_verbose "Would run: go build ${go_flags} -ldflags '${ldflags}' -o ${binary} ."
    return
  fi

  mkdir -p "$bindir"
  log_step "Compiling Go binary"
  if ! go build ${go_flags} -ldflags "${ldflags}" -o "${binary}" .; then
    die "Build failed"
  fi

  chmod +x "$binary" 2>/dev/null || true
  log_success "Built ${binary}"
}

prompt_yes_no() {
  local question="$1"

  if [[ "$DRY_RUN" -eq 1 ]] || [[ ! -t 0 ]]; then
    return 1
  fi

  while true; do
    printf "${BOLD}?${NC} %s [y/N] " "$question"
    local answer
    read -r answer
    answer="${answer,,}"

    case "$answer" in
      y|yes) return 0 ;;
      n|no|"") return 1 ;;
      *) printf "Please answer y(es) or n(o).\n" ;;
    esac
  done
}

setup_inir_alias() {
  local bindir="$1"
  local target="${bindir}/${BINARY_NAME}"
  local alias_path="${bindir}/${ALIAS_NAME}"

  local create_alias=""
  if [[ "$REPLACE_INIR" == "yes" ]]; then
    create_alias="yes"
  elif [[ "$REPLACE_INIR" == "no" ]]; then
    create_alias="no"
  elif prompt_yes_no "Create '${ALIAS_NAME}' symlink pointing to ${BINARY_NAME}?"; then
    create_alias="yes"
  else
    create_alias="no"
  fi

  if [[ "$create_alias" == "no" ]]; then
    log_success "Keeping upstream ${ALIAS_NAME} command"
    return
  fi

  if [[ -L "$alias_path" ]]; then
    local current_target
    current_target="$(readlink -f "$alias_path" 2>/dev/null || true)"
    if [[ "$current_target" == "$target" ]] || [[ "$(basename "$current_target")" == "$BINARY_NAME" ]]; then
      log_success "Symlink already configured: ${ALIAS_NAME} -> ${BINARY_NAME}"
      return
    fi
  fi

  if [[ -e "$alias_path" ]] && [[ ! -L "$alias_path" ]]; then
    local backup="${alias_path}.upstream.bak"
    if [[ "$DRY_RUN" -eq 1 ]]; then
      log_verbose "Would back up ${alias_path} to ${backup}"
      log_verbose "Would create symlink ${alias_path} -> ${target}"
      return
    fi
    mv "$alias_path" "$backup"
    log_warn "Backed up upstream ${ALIAS_NAME} to ${backup}"
  fi

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log_verbose "Would create symlink ${alias_path} -> ${target}"
    return
  fi

  ln -sfn "$target" "$alias_path"
  log_success "Symlink created: ${ALIAS_NAME} -> ${BINARY_NAME}"
}

ensure_path() {
  local bindir="$1"
  if [[ ":$PATH:" == *":${bindir}:"* ]]; then
    log_success "${bindir} is already on PATH"
    return
  fi

  log_warn "${bindir} is not on PATH"

  local shell_name rc_file export_line
  shell_name="$(basename "${SHELL:-bash}")"

  case "$shell_name" in
    bash)
      rc_file="$HOME/.bashrc"
      export_line="export PATH=\"${bindir}:\$PATH\""
      ;;
    zsh)
      rc_file="$HOME/.zshrc"
      export_line="export PATH=\"${bindir}:\$PATH\""
      ;;
    fish)
      rc_file="$HOME/.config/fish/config.fish"
      export_line="set -gx PATH ${bindir} \$PATH"
      ;;
    *)
      log_warn "Unknown shell '${shell_name}'. Add ${bindir} to PATH manually."
      return
      ;;
  esac

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log_verbose "Would append PATH entry to ${rc_file}"
    return
  fi

  if [[ -f "$rc_file" ]] && grep -q "${bindir}" "$rc_file" 2>/dev/null; then
    log_info "PATH entry already exists in ${rc_file}"
    return
  fi

  {
    printf "\n# Added by inir-cli installer\n"
    printf "%s\n" "$export_line"
  } >> "$rc_file"

  log_success "Updated PATH in ${rc_file}"
}

verify_install() {
  local bindir="$1"
  local binary="${bindir}/${BINARY_NAME}"

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log_verbose "Would verify ${binary} --help"
    return
  fi

  [[ -x "$binary" ]] || die "Binary not found at ${binary}"

  if "$binary" --help >/dev/null 2>&1; then
    log_success "Verification passed"
  else
    log_warn "Binary exists but --help returned an error"
  fi

  local version_output
  version_output="$("$binary" --version 2>/dev/null || printf 'unknown')"
  log_info "Installed version: ${version_output}"
}

setup_systemd_service() {
  local bindir="$1"
  local binary="${bindir}/${BINARY_NAME}"

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log_verbose "Would run: ${binary} service install"
    log_verbose "Would run: ${binary} service enable"
    return
  fi

  if ! command -v systemctl >/dev/null 2>&1; then
    log_verbose "systemctl not found, skipping user service setup"
    return
  fi

  if ! systemctl --user show-environment >/dev/null 2>&1; then
    log_verbose "systemd --user not reachable, skipping user service setup"
    return
  fi

  if "$binary" service install >/dev/null 2>&1; then
    log_success "Installed inir user service"
  else
    log_warn "Failed to install inir user service (non-fatal)"
    return
  fi

  if "$binary" service enable >/dev/null 2>&1; then
    log_success "Enabled inir user service"
  else
    log_warn "Could not enable inir user service (non-fatal)"
  fi
}

do_uninstall() {
  local bindir="$1"
  local binary="${bindir}/${BINARY_NAME}"
  local alias_path="${bindir}/${ALIAS_NAME}"
  local backup="${alias_path}.upstream.bak"

  if [[ -f "$binary" ]]; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
      log_verbose "Would remove ${binary}"
    else
      rm -f "$binary"
      log_success "Removed ${binary}"
    fi
  else
    log_warn "${binary} not found"
  fi

  if [[ -L "$alias_path" ]]; then
    local alias_target
    alias_target="$(readlink -f "$alias_path" 2>/dev/null || true)"
    if [[ "$(basename "$alias_target")" == "$BINARY_NAME" ]]; then
      if [[ "$DRY_RUN" -eq 1 ]]; then
        log_verbose "Would remove symlink ${alias_path}"
      else
        rm -f "$alias_path"
        log_success "Removed ${ALIAS_NAME} symlink"
      fi
    fi
  fi

  if [[ -f "$backup" ]]; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
      log_verbose "Would restore ${backup} to ${alias_path}"
    else
      mv "$backup" "$alias_path"
      log_success "Restored upstream ${ALIAS_NAME} from backup"
    fi
  fi
}

print_post_install() {
  local bindir="$1"
  local alias_path="${bindir}/${ALIAS_NAME}"

  printf "\n${GREEN}+------------------------------------------------------------------+${NC}\n"
  printf "${GREEN}|                    Installation Complete                         |${NC}\n"
  printf "${GREEN}+------------------------------------------------------------------+${NC}\n\n"

  printf "${BOLD}Quick start:${NC}\n"
  if [[ -L "$alias_path" ]]; then
    printf "  %sinir generate --image /path/to/wallpaper.jpg%s\n" "$CYAN" "$NC"
    printf "  %sinir scheme --list%s\n" "$CYAN" "$NC"
    printf "  %sinir theme apply all%s\n" "$CYAN" "$NC"
  else
    printf "  %sinir-cli generate --image /path/to/wallpaper.jpg%s\n" "$CYAN" "$NC"
    printf "  %sinir-cli scheme --list%s\n" "$CYAN" "$NC"
    printf "  %sinir-cli theme apply all%s\n" "$CYAN" "$NC"
  fi

  printf "\n${BOLD}Docs:${NC}\n"
  printf "  %shttps://github.com/yukazakiri/inir-cli%s\n\n" "$CYAN" "$NC"
}

main() {
  local bindir
  bindir="$(resolve_bindir)"

  print_header
  log_info "Install directory: ${bindir}"

  if [[ "$UNINSTALL" -eq 1 ]]; then
    log_info "Starting uninstall"
    do_uninstall "$bindir"
    log_success "Uninstall complete"
    exit 0
  fi

  detect_go
  build_binary "$bindir"
  setup_inir_alias "$bindir"
  ensure_path "$bindir"
  verify_install "$bindir"
  setup_systemd_service "$bindir"
  print_post_install "$bindir"
}

main "$@"
