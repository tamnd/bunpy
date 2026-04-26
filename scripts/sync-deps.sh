#!/usr/bin/env bash
# Clones the sibling toolchain repos that bunpy composes:
# gopapy (parser), gocopy (compiler), goipy (bytecode VM).
#
# bunpy depends on these via a go.work workspace. The workspace
# wires in local clones in the same parent directory. CI calls
# this script before running tests so the workspace resolves.
#
# Pinned commits are listed below. Update them with intent: every
# bump should be paired with a tested CI run.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PARENT="$(dirname "$ROOT")"

GOPAPY_REPO="https://github.com/tamnd/gopapy.git"
GOCOPY_REPO="https://github.com/tamnd/gocopy.git"
GOIPY_REPO="https://github.com/tamnd/goipy.git"

GOPAPY_REV="df586985862310a92808e7963e1eaa30f7b48631"
GOCOPY_REV="adfe1903651080d236a7ecde9086d2804622214c"
GOIPY_REV="e6b4eda3e67fa197ac0f115b4bef4a264f2540bd"

clone_at_rev() {
  local repo="$1"
  local dest="$2"
  local rev="$3"

  if [ -d "$dest/.git" ]; then
    echo "sync-deps: $dest already cloned, fetching"
    git -C "$dest" fetch --quiet origin "$rev" || git -C "$dest" fetch --quiet origin
  else
    echo "sync-deps: cloning $repo into $dest"
    git clone --quiet "$repo" "$dest"
  fi
  git -C "$dest" -c advice.detachedHead=false checkout --quiet "$rev"
}

clone_at_rev "$GOPAPY_REPO" "$PARENT/gopapy" "$GOPAPY_REV"
clone_at_rev "$GOCOPY_REPO" "$PARENT/gocopy" "$GOCOPY_REV"
clone_at_rev "$GOIPY_REPO"  "$PARENT/goipy"  "$GOIPY_REV"

echo "sync-deps: ready"
