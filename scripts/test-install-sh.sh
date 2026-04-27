#!/usr/bin/env bash
# Smoke-test install.sh against the latest published release.
# Skips on CI when network is unavailable or arch is unsupported.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# install.sh only ships linux+darwin binaries.
case "$(uname -s)" in
  Linux|Darwin) ;;
  *) echo "test-install-sh: skipping on $(uname -s) (linux/darwin only)"; exit 0 ;;
esac

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

if ! BUNPY_INSTALL_DIR="$tmpdir" bash "$ROOT/install.sh" >/dev/null 2>"$tmpdir/log"; then
  cat "$tmpdir/log" >&2
  echo "test-install-sh: installer failed" >&2
  exit 1
fi

if [ ! -x "$tmpdir/bin/bunpy" ]; then
  echo "test-install-sh: $tmpdir/bin/bunpy missing or not executable" >&2
  exit 1
fi

short="$("$tmpdir/bin/bunpy" version --short)"
case "$short" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "test-install-sh: unexpected --short output: $short" >&2; exit 1 ;;
esac

echo "test-install-sh: ok ($short)"
