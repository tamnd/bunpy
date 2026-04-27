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
# The pinned sibling commits (gopapy, gocopy, goipy) come from the
# constants in scripts/sync-deps.sh, so there's one source of truth.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYNC="$ROOT/scripts/sync-deps.sh"

version="${BUNPY_VERSION:-dev}"
commit="${BUNPY_COMMIT:-}"
if [ -z "$commit" ]; then
  commit="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo "")"
fi
date="${BUNPY_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

extract_rev() {
  local key="$1"
  awk -v k="^${key}=" '$0 ~ k {
    s = $0
    sub(k, "", s)
    gsub(/"/, "", s)
    print substr(s, 1, 7)
    exit
  }' "$SYNC"
}

gopapy="$(extract_rev GOPAPY_REV)"
gocopy="$(extract_rev GOCOPY_REV)"
goipy="$(extract_rev GOIPY_REV)"

pkg="github.com/tamnd/bunpy/v1/runtime"
printf -- "-s -w"
printf -- " -X %s.Version=%s" "$pkg" "$version"
printf -- " -X %s.Commit=%s" "$pkg" "$commit"
printf -- " -X %s.BuildDate=%s" "$pkg" "$date"
printf -- " -X %s.Goipy=%s" "$pkg" "$goipy"
printf -- " -X %s.Gocopy=%s" "$pkg" "$gocopy"
printf -- " -X %s.Gopapy=%s" "$pkg" "$gopapy"
printf -- "\n"
