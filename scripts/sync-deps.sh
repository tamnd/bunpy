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
DEPS="$ROOT/.deps"

GOPAPY_REPO="https://github.com/tamnd/gopapy.git"
GOCOPY_REPO="https://github.com/tamnd/gocopy.git"
GOIPY_REPO="https://github.com/tamnd/goipy.git"

GOPAPY_REV="7e162ceac871cdefb7e25cecf9b10504c0660cc9"
GOCOPY_REV="bdaac9f48f29e08a327118916c511af216940623"
GOIPY_REV="2a20bf907ec38c5870087afd7b72607f63ca73b0"

mkdir -p "$DEPS"

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
  git -C "$dest" -c advice.detachedHead=false checkout --quiet --force "$rev"
}

clone_at_rev "$GOPAPY_REPO" "$DEPS/gopapy" "$GOPAPY_REV"
clone_at_rev "$GOCOPY_REPO" "$DEPS/gocopy" "$GOCOPY_REV"
clone_at_rev "$GOIPY_REPO"  "$DEPS/goipy"  "$GOIPY_REV"

echo "sync-deps: ready"
