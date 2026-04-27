#!/usr/bin/env bash
# render-formula.sh: render Formula/bunpy.rb from
# scripts/formula.rb.tmpl plus a SHA256SUMS file.
#
# Usage:
#   render-formula.sh <version> <sha256sums-file> <out-file>
#
# version is bare ("0.0.6", no leading v). The sha256sums file is
# the aggregated checksum file the release workflow uploads, with
# lines of the form:
#
#   <sha256>  bunpy-vX.Y.Z-<os>-<arch>.tar.gz
#
# Only the four supported {darwin,linux} x {amd64,arm64} entries
# are read. Windows .zip lines are ignored: Homebrew does not ship
# Windows binaries.
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: render-formula.sh <version> <sha256sums-file> <out-file>" >&2
  exit 2
fi

version="$1"
sums="$2"
out="$3"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
tmpl="$ROOT/scripts/formula.rb.tmpl"
[ -f "$tmpl" ] || { echo "missing template: $tmpl" >&2; exit 1; }
[ -f "$sums" ] || { echo "missing sums: $sums" >&2; exit 1; }

extract() {
  local archive="bunpy-v${version}-$1-$2.tar.gz"
  local sha
  sha="$(awk -v f="$archive" '$2 == f { print $1; exit }' "$sums")"
  if [ -z "$sha" ]; then
    echo "render-formula: no sha256 for $archive in $sums" >&2
    exit 1
  fi
  printf '%s' "$sha"
}

darwin_arm64="$(extract darwin arm64)"
darwin_amd64="$(extract darwin amd64)"
linux_arm64="$(extract linux arm64)"
linux_amd64="$(extract linux amd64)"

sed \
  -e "s|@@VERSION@@|${version}|g" \
  -e "s|@@SHA_DARWIN_ARM64@@|${darwin_arm64}|g" \
  -e "s|@@SHA_DARWIN_AMD64@@|${darwin_amd64}|g" \
  -e "s|@@SHA_LINUX_ARM64@@|${linux_arm64}|g" \
  -e "s|@@SHA_LINUX_AMD64@@|${linux_amd64}|g" \
  "$tmpl" > "$out"

echo "rendered $out"
