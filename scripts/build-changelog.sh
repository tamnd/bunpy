#!/usr/bin/env bash
#
# build-changelog.sh — concatenate per-version changelog/v*.md notes into
# CHANGELOG.md, sorted newest first. Inspired by gopapy's tooling.
#
# Each changelog/vX.Y.Z.md contains the body that ships in the GitHub
# release notes (release.yml feeds it via body_path). This script glues
# those bodies into the top-level CHANGELOG.md so the repo always shows a
# single, current view of what landed in each release.

set -euo pipefail

cd "$(dirname "$0")/.."

out=CHANGELOG.md
tmp=$(mktemp)

cat > "$tmp" <<'EOF'
# Changelog

All notable changes to bunpy are recorded here. The format follows
[Keep a Changelog 1.1](https://keepachangelog.com/en/1.1.0/). Once
bunpy reaches 1.0 the project will follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html); until
then, expect minor version bumps to sometimes include breaking
changes.

## [Unreleased]

EOF

# List release notes newest version first. `sort -V -r` handles semver
# without dragging in extra deps; missing files just yield an empty body.
shopt -s nullglob
notes=( changelog/v*.md )

if [ ${#notes[@]} -gt 0 ]; then
  # Reverse-sort by version.
  IFS=$'\n' sorted=( $(printf '%s\n' "${notes[@]}" | sort -V -r) )
  unset IFS
  for f in "${sorted[@]}"; do
    cat "$f" >> "$tmp"
    printf '\n' >> "$tmp"
  done
fi

mv "$tmp" "$out"
echo "wrote $out"
