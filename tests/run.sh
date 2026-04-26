#!/usr/bin/env bash
# End-to-end harness: builds bunpy, then walks tests/fixtures/v00*
# running each .py and comparing stdout against the matching
# expected_<name>.txt. Exit 0 only if every fixture matches.
#
# Used by CI on linux, macOS, and Windows (via git-bash).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

bin="$(mktemp -d)/bunpy"
case "$(uname -s 2>/dev/null || echo Windows)" in
  MINGW*|MSYS*|CYGWIN*|Windows*) bin="${bin}.exe" ;;
esac

echo "build: $bin"
go build -o "$bin" ./cmd/bunpy

fail=0
ran=0

run_fixture() {
  local script="$1"
  local dir name expected
  dir="$(dirname "$script")"
  name="$(basename "$script" .py)"
  expected="$dir/expected_${name}.txt"

  if [ ! -f "$expected" ]; then
    echo "skip: $script (no expected_${name}.txt)"
    return 0
  fi

  ran=$((ran + 1))
  local got rc
  got="$("$bin" "$script" 2>&1)" || rc=$?
  rc="${rc:-0}"
  local want
  want="$(cat "$expected")"
  if [ "$rc" -ne 0 ]; then
    echo "FAIL: $script exited $rc"
    echo "stderr+stdout:"
    echo "$got"
    fail=$((fail + 1))
    return 0
  fi
  if [ "$got" != "$want" ]; then
    echo "FAIL: $script stdout mismatch"
    echo "  got:  $(printf '%s' "$got" | head -c 200)"
    echo "  want: $(printf '%s' "$want" | head -c 200)"
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   $script"
}

for script in tests/fixtures/v00*/*.py; do
  [ -e "$script" ] || continue
  run_fixture "$script"
done

echo "---"
echo "ran $ran fixtures, $fail failed"
exit "$fail"
