#!/usr/bin/env bash
#
# nowshowing-pp-cli fleet installer — idempotent, macOS + Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/ph-commons/nowshowing-pp-cli/master/scripts/install.sh | bash
#
# Prefers the prebuilt GitHub release (no local modernc.org/sqlite compile).
# Falls back to `go install` only when the download cannot be resolved.
# If this machine has the ngpestelos fleet layout (~/src/hermes-config),
# also wires the nowshowing-pp-cli skill; skipped cleanly elsewhere.

set -euo pipefail

MODULE="github.com/ph-commons/nowshowing-pp-cli"
BIN="nowshowing-pp-cli"
GOBIN_DIR="${GOBIN:-$HOME/.local/bin}"
OWNER_REPO="ph-commons/nowshowing-pp-cli"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarn:\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

# Verify a downloaded release tarball's SHA-256 against goreleaser's published
# checksums.txt before any extraction. Fails closed (die) on any problem:
# unreachable checksums file, no entry for this tarball, hash mismatch, or no
# hashing tool available. Does NOT fall back to source build on failure here —
# that fallback exists for network/availability problems, not for a security
# signal (see issue #7).
verify_checksum() {
  local tmp_dir="$1" tarball_name="$2" checksums_url="$3"
  local checksums_file="$tmp_dir/checksums.txt"

  if ! curl -fsSL "$checksums_url" -o "$checksums_file" 2>/dev/null; then
    die "checksum verification failed: could not download $checksums_url (refusing to install unverified binary)"
  fi

  local expected
  # `|| true` is required here, not optional style: under `set -euo pipefail`,
  # a no-match `grep -F` exits 1, `pipefail` propagates that to the whole
  # pipeline, and this sits inside a plain `var=$(...)` assignment (not an
  # `if`/`&&`/`||`), which is NOT set-e-exempt. Without `|| true` the shell
  # exits right here on a "no entry" tarball, before the die() below ever
  # runs, and the caller gets a silent, unexplained hang/stop with no
  # diagnostic instead of the intended clear error message.
  expected="$(grep -F "  ${tarball_name}" "$checksums_file" 2>/dev/null | awk '{print $1}' | head -1 || true)"
  [ -n "$expected" ] || die "checksum verification failed: no entry for $tarball_name in checksums.txt (refusing to install unverified binary)"

  local actual
  # Same `|| true` reasoning as the `expected=` assignment above: if the
  # tarball vanished/unreadable between download and here, sha256sum/shasum
  # exits nonzero, pipefail propagates it into this assignment, and without
  # `|| true` the shell would exit here with a raw tool error instead of the
  # intended die() message below.
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmp_dir/$tarball_name" 2>/dev/null | awk '{print $1}' || true)"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$tmp_dir/$tarball_name" 2>/dev/null | awk '{print $1}' || true)"
  else
    die "checksum verification failed: neither sha256sum nor shasum is available (refusing to install unverified binary)"
  fi
  [ -n "$actual" ] || die "checksum verification failed: unable to hash $tarball_name (file missing or unreadable)"

  [ "$expected" = "$actual" ] || die "checksum MISMATCH for $tarball_name: expected $expected, got $actual (refusing to install — release asset may be tampered)"
  log "Checksum verified: $tarball_name"
}

mkdir -p "$GOBIN_DIR"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) arch="" ;;
esac

install_ok=false
if [ -n "$arch" ] && command -v curl >/dev/null 2>&1; then
  ver="$(curl -fsSL "https://api.github.com/repos/${OWNER_REPO}/releases/latest" 2>/dev/null \
    | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -1)"
  if [ -n "$ver" ]; then
    tarball="nowshowing-pp-cli_${ver}_${os}_${arch}.tar.gz"
    url="https://github.com/${OWNER_REPO}/releases/download/v${ver}/${tarball}"
    log "Downloading prebuilt $BIN v$ver ($os/$arch)"
    tmp="$(mktemp -d)"
    if curl -fsSL "$url" -o "$tmp/$tarball" 2>/dev/null; then
      checksums_url="https://github.com/${OWNER_REPO}/releases/download/v${ver}/checksums.txt"
      verify_checksum "$tmp" "$tarball" "$checksums_url"
      if tar -xzf "$tmp/$tarball" -C "$GOBIN_DIR" 2>/dev/null; then
        chmod +x "$GOBIN_DIR/nowshowing-pp-cli" "$GOBIN_DIR/nowshowing-pp-mcp" 2>/dev/null || true
        install_ok=true
      else
        warn "extraction failed after checksum verification; will try building from source."
      fi
    else
      warn "prebuilt download failed ($url); will try building from source."
    fi
    rm -rf "$tmp"
  fi
fi

if [ "$install_ok" != true ]; then
  command -v go >/dev/null 2>&1 || die "No prebuilt binary available and Go not on PATH. Install Go 1.21+ (https://go.dev/dl/) or check release assets."
  warn "Building from source — this compiles modernc.org/sqlite and is CPU-heavy."
  for attempt in 1 2 3; do
    if GOTOOLCHAIN=auto GOBIN="$GOBIN_DIR" go install "${MODULE}/cmd/${BIN}@latest" 2>/tmp/nowshowing-install.err; then
      install_ok=true
      break
    fi
    if grep -q "sum.golang.org" /tmp/nowshowing-install.err 2>/dev/null; then
      warn "checksum DB not ready (attempt $attempt/3); retrying in 10s"
      sleep 10
    else
      cat /tmp/nowshowing-install.err >&2
      break
    fi
  done
  rm -f /tmp/nowshowing-install.err
fi
[ "$install_ok" = true ] || die "install failed (neither prebuilt download nor go install worked)."

log "Installed: $($GOBIN_DIR/$BIN --version 2>/dev/null || echo "$GOBIN_DIR/$BIN")"
case ":$PATH:" in
  *":$GOBIN_DIR:"*) ;;
  *) warn "$GOBIN_DIR is not on PATH — add it to use $BIN" ;;
esac

# Fleet skill wiring (optional)
SKILL_SRC="${HERMES_CONFIG:-$HOME/src/hermes-config}/skills/pp-nowshowing"
if [ -d "$SKILL_SRC" ] && [ -d "$HOME/.claude/skills" ]; then
  ln -sfn "$SKILL_SRC" "$HOME/.claude/skills/pp-nowshowing"
  log "Linked ~/.claude/skills/pp-nowshowing"
fi

log "Smoke: $BIN theaters --json"
"$GOBIN_DIR/$BIN" theaters --json >/dev/null || warn "session smoke failed (network?)"
log "Done."
