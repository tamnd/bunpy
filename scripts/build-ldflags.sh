#!/usr/bin/env bash
# build-ldflags.sh: print the -ldflags string used to bake metadata
# into bunpy. The build pipeline (build.yml, release.yml) and the CI
# smoke job all source this so the metadata story is consistent.
#
# Inputs come from env vars or, where missing, sane defaults:
#   BUNPY_VERSION     defaults to "dev"
#   BUNPY_COMMIT      defaults to `git rev-parse --short HEAD` if available
#   BUNPY_BUILD_DATE  defaults to current UTC time, RFC 3339
#
# The toolchain versions (gopapy, gocopy, goipy) are read from go.mod.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOMOD="$ROOT/go.mod"

version="${BUNPY_VERSION:-dev}"
commit="${BUNPY_COMMIT:-}"
if [ -z "$commit" ]; then
  commit="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo "")"
fi
date="${BUNPY_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

# Read module version from go.mod, strip leading 'v' and return first 7 chars.
extract_mod() {
  local mod="$1"
  awk -v m="$mod" '$1 == m { v=$2; sub(/^v/,"",v); print substr(v,1,7); exit }' "$GOMOD"
}

gopapy="$(extract_mod github.com/tamnd/gopapy)"
gocopy="$(extract_mod github.com/tamnd/gocopy)"
goipy="$(extract_mod github.com/tamnd/goipy)"

pkg="github.com/tamnd/bunpy/v1/runtime"
printf -- "-s -w"
printf -- " -X %s.Version=%s" "$pkg" "$version"
printf -- " -X %s.Commit=%s" "$pkg" "$commit"
printf -- " -X %s.BuildDate=%s" "$pkg" "$date"
printf -- " -X %s.Goipy=%s" "$pkg" "$goipy"
printf -- " -X %s.Gocopy=%s" "$pkg" "$gocopy"
printf -- " -X %s.Gopapy=%s" "$pkg" "$gopapy"
printf -- "\n"
