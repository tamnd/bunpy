#!/usr/bin/env bash
# Smoke-test scripts/render-formula.sh against a fixed sums fixture.
# Run by tests/run.sh and CI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/sums" <<'EOF'
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  bunpy-v0.0.6-darwin-amd64.tar.gz
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  bunpy-v0.0.6-darwin-arm64.tar.gz
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  bunpy-v0.0.6-linux-amd64.tar.gz
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  bunpy-v0.0.6-linux-arm64.tar.gz
eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee  bunpy-v0.0.6-windows-amd64.zip
ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff  bunpy-v0.0.6-windows-arm64.zip
EOF

bash "$ROOT/scripts/render-formula.sh" 0.0.6 "$tmp/sums" "$tmp/bunpy.rb" >/dev/null

want_strings=(
  'version "0.0.6"'
  'sha256 "aaaaaaaa'
  'sha256 "bbbbbbbb'
  'sha256 "cccccccc'
  'sha256 "dddddddd'
)
for s in "${want_strings[@]}"; do
  if ! grep -q -F "$s" "$tmp/bunpy.rb"; then
    echo "render-formula: missing expected substring: $s" >&2
    cat "$tmp/bunpy.rb" >&2
    exit 1
  fi
done

# Windows shas must NOT appear: the formula does not ship them.
for s in eeeeeeee ffffffff; do
  if grep -q "$s" "$tmp/bunpy.rb"; then
    echo "render-formula: leaked windows sha into formula: $s" >&2
    exit 1
  fi
done

# Missing sum should fail.
cat > "$tmp/sums-incomplete" <<'EOF'
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  bunpy-v0.0.6-darwin-amd64.tar.gz
EOF
if bash "$ROOT/scripts/render-formula.sh" 0.0.6 "$tmp/sums-incomplete" "$tmp/should-not-exist.rb" >/dev/null 2>&1; then
  echo "render-formula: should have failed on incomplete sums" >&2
  exit 1
fi

echo "render-formula: ok"
