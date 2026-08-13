#!/bin/sh
# backupctl-install.sh — one-line installer for backupctl.
#
#   curl -fsSL https://raw.githubusercontent.com/Solo-it/backupctl/main/backupctl-install.sh | sh
#
# What it does, depending on what's available:
#   - macOS with Homebrew:      brew tap + brew trust + brew install
#   - Debian/Ubuntu with apt:   imports the GPG signing key, adds the apt
#                                repo, apt install (see SECURITY.md)
#   - anything else:            downloads the matching binary from the
#                                latest GitHub release and verifies its
#                                SHA256 against checksums.txt before
#                                installing it to /usr/local/bin
#
# This script does exactly what the README's manual steps do — it's not a
# shortcut around verification, everything downloaded is still checked.
set -eu

REPO="Solo-it/backupctl"
BIN_NAME="backupctl"

log() { printf '==> %s\n' "$1"; }
die() { printf 'error: %s\n' "$1" >&2; exit 1; }

os="$(uname -s)"

if [ "$os" = "Darwin" ] && command -v brew >/dev/null 2>&1; then
    log "Homebrew detected — installing via Solo-it/tap"
    brew tap Solo-it/tap
    brew trust solo-it/tap 2>/dev/null || true # older brew versions don't have/need this
    brew install "$BIN_NAME"
    log "done: $(command -v backupctl)"
    exit 0
fi

if [ "$os" = "Linux" ] && command -v apt-get >/dev/null 2>&1; then
    log "apt detected — setting up the signed backupctl apt repository"
    [ "$(id -u)" -eq 0 ] || die "this branch needs root (apt/gpg keyring) — re-run with sudo"
    curl -fsSL https://solo-it.github.io/backupctl/apt/signing-key.asc | gpg --dearmor -o /usr/share/keyrings/backupctl.gpg
    echo "deb [signed-by=/usr/share/keyrings/backupctl.gpg] https://solo-it.github.io/backupctl/apt stable main" > /etc/apt/sources.list.d/backupctl.list
    apt-get update
    apt-get install -y "$BIN_NAME"
    log "done: $(command -v backupctl)"
    exit 0
fi

# Fallback: download the matching binary archive from the latest GitHub
# release and verify it against checksums.txt before installing.
log "no apt/brew found — falling back to a direct binary install"

arch="$(uname -m)"
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) die "unsupported architecture: $arch — download manually from https://github.com/$REPO/releases" ;;
esac

case "$os" in
    Linux) goos=linux; ext=tar.gz ;;
    Darwin) goos=darwin; ext=tar.gz ;;
    *) die "unsupported OS: $os — download manually from https://github.com/$REPO/releases" ;;
esac

api_url="https://api.github.com/repos/$REPO/releases/latest"
version="$(curl -fsSL "$api_url" | grep -m1 '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
[ -n "$version" ] || die "could not determine the latest release from $api_url"
version_num="${version#v}"

archive="${BIN_NAME}_${version_num}_${goos}_${arch}.${ext}"
base_url="https://github.com/$REPO/releases/download/$version"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

log "downloading $archive ($version)"
curl -fsSL -o "$tmpdir/$archive" "$base_url/$archive"
curl -fsSL -o "$tmpdir/checksums.txt" "$base_url/checksums.txt"

log "verifying checksum"
( cd "$tmpdir" && grep " $archive\$" checksums.txt | sha256sum -c - ) \
    || die "checksum verification failed — not installing"

tar -xzf "$tmpdir/$archive" -C "$tmpdir" "$BIN_NAME"

install_dir="/usr/local/bin"
if [ ! -w "$install_dir" ] 2>/dev/null; then
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
    log "no write access to /usr/local/bin, installing to $install_dir instead (make sure it's in PATH)"
fi

install -m 755 "$tmpdir/$BIN_NAME" "$install_dir/$BIN_NAME"
log "done: $install_dir/$BIN_NAME"
