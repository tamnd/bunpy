#!/usr/bin/env bash
# install.sh: download and install bunpy from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/tamnd/bunpy/main/install.sh | bash
#
# Env overrides:
#   BUNPY_VERSION       pin to a tag (default: latest from GitHub API)
#   BUNPY_INSTALL_DIR   install root (default: $HOME/.bunpy)
#   BUNPY_OS            override detected os (linux, darwin)
#   BUNPY_ARCH          override detected arch (amd64, arm64)
#
# The script verifies the SHA-256 checksum from the release before
# installing. Re-running upgrades in place; the previous binary is
# saved as bin/bunpy.prev so a rollback is one mv away.
set -euo pipefail

REPO="tamnd/bunpy"
INSTALL_DIR="${BUNPY_INSTALL_DIR:-$HOME/.bunpy}"

err() { echo "install.sh: $*" >&2; exit 1; }
note() { echo "install.sh: $*"; }

need() {
  command -v "$1" >/dev/null 2>&1 || err "missing required tool: $1"
}

need curl
need tar
if command -v shasum >/dev/null 2>&1; then
  sha_cmd="shasum -a 256"
elif command -v sha256sum >/dev/null 2>&1; then
  sha_cmd="sha256sum"
else
  err "missing sha256 tool (need shasum or sha256sum)"
fi

os="${BUNPY_OS:-}"
if [ -z "$os" ]; then
  case "$(uname -s)" in
    Linux)  os=linux ;;
    Darwin) os=darwin ;;
    *)      err "unsupported os: $(uname -s). Windows users: download the .zip from releases." ;;
  esac
fi

arch="${BUNPY_ARCH:-}"
if [ -z "$arch" ]; then
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) err "unsupported arch: $(uname -m)" ;;
  esac
fi

version="${BUNPY_VERSION:-}"
if [ -z "$version" ]; then
  note "resolving latest release"
  version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' \
    | head -n1)"
  [ -n "$version" ] || err "could not resolve latest release"
fi
case "$version" in
  v*) ;;
  *) version="v${version}" ;;
esac

archive="bunpy-${version}-${os}-${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${archive}"
sums_url="https://github.com/${REPO}/releases/download/${version}/SHA256SUMS"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

note "downloading ${archive}"
curl -fsSL "$url" -o "${tmp}/${archive}"

note "downloading SHA256SUMS"
curl -fsSL "$sums_url" -o "${tmp}/SHA256SUMS"

note "verifying checksum"
( cd "$tmp" && grep " ${archive}\$" SHA256SUMS > "${archive}.sha256" \
    && $sha_cmd -c "${archive}.sha256" )

note "extracting"
tar -xzf "${tmp}/${archive}" -C "$tmp"

mkdir -p "${INSTALL_DIR}/bin"
inner_dir="${tmp}/bunpy-${version}-${os}-${arch}"
[ -d "$inner_dir" ] || err "extract layout unexpected: $inner_dir missing"

if [ -e "${INSTALL_DIR}/bin/bunpy" ]; then
  mv -f "${INSTALL_DIR}/bin/bunpy" "${INSTALL_DIR}/bin/bunpy.prev"
fi

cp "${inner_dir}/bunpy" "${INSTALL_DIR}/bin/bunpy"
chmod +x "${INSTALL_DIR}/bin/bunpy"

note "installed bunpy ${version} to ${INSTALL_DIR}/bin/bunpy"
echo
echo "Add this to your shell rc if you have not already:"
echo "  export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
echo
"${INSTALL_DIR}/bin/bunpy" version || true
