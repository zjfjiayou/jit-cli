#!/bin/sh
set -eu

REPO="${JIT_CLI_REPO:-wanyun/JitCli}"
BIN_NAME="${JIT_CLI_BIN_NAME:-jit}"
INSTALL_DIR="${JIT_CLI_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${JIT_CLI_VERSION:-latest}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

fail() {
  printf '%s\n' "$*" >&2
  exit 1
}

download() {
  url="$1"
  dest="$2"
  if need_cmd curl; then
    curl -fsSL "$url" -o "$dest"
    return
  fi
  if need_cmd wget; then
    wget -qO "$dest" "$url"
    return
  fi
  fail "curl or wget is required"
}

detect_os() {
  case "$(uname -s)" in
    Linux*) printf "linux" ;;
    Darwin*) printf "darwin" ;;
    *) fail "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf "amd64" ;;
    arm64|aarch64) printf "arm64" ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [ "$VERSION" != "latest" ]; then
    printf '%s' "$VERSION"
    return
  fi
  if need_cmd curl; then
    resolved="$(curl -fsSI "https://github.com/${REPO}/releases/latest" | tr -d '\r' | awk -F'/' '/^location:/ {print $NF}' | tail -n1)"
  else
    resolved="$(wget --server-response --max-redirect=0 -O /dev/null "https://github.com/${REPO}/releases/latest" 2>&1 | tr -d '\r' | awk -F'/' '/Location:/ {print $NF}' | tail -n1)"
  fi
  [ -n "$resolved" ] || fail "failed to resolve latest version, set JIT_CLI_VERSION explicitly"
  printf '%s' "$resolved"
}

main() {
  os="$(detect_os)"
  arch="$(detect_arch)"
  version_tag="$(resolve_version)"

  archive="${BIN_NAME}-${os}-${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${version_tag}/${archive}"

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM
  archive_path="${tmpdir}/${archive}"

  printf 'Downloading %s %s (%s/%s)\n' "$BIN_NAME" "$version_tag" "$os" "$arch"
  download "$url" "$archive_path"

  mkdir -p "$tmpdir/extract" "$INSTALL_DIR"
  tar -xzf "$archive_path" -C "$tmpdir/extract"

  extracted="${tmpdir}/extract/${BIN_NAME}"
  [ -f "$extracted" ] || fail "binary not found in archive: $archive"

  install -m 0755 "$extracted" "${INSTALL_DIR}/${BIN_NAME}"
  printf 'Installed to %s\n' "${INSTALL_DIR}/${BIN_NAME}"
  printf 'Ensure "%s" is in your PATH\n' "$INSTALL_DIR"
}

main "$@"
