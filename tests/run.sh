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
  local label="$1"
  local script="$2"
  shift 2
  local dir name expected
  dir="$(dirname "$script")"
  name="$(basename "$script" .py)"
  expected="$dir/expected_${name}.txt"

  if [ ! -f "$expected" ]; then
    echo "skip: $label $script (no expected_${name}.txt)"
    return 0
  fi

  ran=$((ran + 1))
  local got rc
  got="$("$bin" "$@" "$script" 2>&1)" || rc=$?
  rc="${rc:-0}"
  local want
  want="$(cat "$expected")"
  if [ "$rc" -ne 0 ]; then
    echo "FAIL: $label $script exited $rc"
    echo "stderr+stdout:"
    echo "$got"
    fail=$((fail + 1))
    return 0
  fi
  if [ "$got" != "$want" ]; then
    echo "FAIL: $label $script stdout mismatch"
    echo "  got:  $(printf '%s' "$got" | head -c 200)"
    echo "  want: $(printf '%s' "$want" | head -c 200)"
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   $label $script"
}

for script in tests/fixtures/v00*/*.py; do
  [ -e "$script" ] || continue
  run_fixture "positional" "$script"
  run_fixture "run       " "$script" run
done

run_repl_fixture() {
  local input="$1"
  local dir name expected got rc want
  dir="$(dirname "$input")"
  name="$(basename "$input" .repl_in)"
  expected="$dir/expected_${name}.txt"

  if [ ! -f "$expected" ]; then
    echo "skip: repl       $input (no expected_${name}.txt)"
    return 0
  fi

  ran=$((ran + 1))
  got="$("$bin" repl --quiet < "$input" 2>&1)" || rc=$?
  rc="${rc:-0}"
  want="$(cat "$expected")"
  if [ "$rc" -ne 0 ]; then
    echo "FAIL: repl       $input exited $rc"
    echo "stderr+stdout:"
    echo "$got"
    fail=$((fail + 1))
    return 0
  fi
  if [ "$got" != "$want" ]; then
    echo "FAIL: repl       $input stdout mismatch"
    echo "  got:  $(printf '%s' "$got" | head -c 200)"
    echo "  want: $(printf '%s' "$want" | head -c 200)"
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   repl       $input"
}

for input in tests/fixtures/v00*/*.repl_in; do
  [ -e "$input" ] || continue
  run_repl_fixture "$input"
done

run_pm_config_fixture() {
  local toml="$1"
  local dir name expected got rc want
  dir="$(dirname "$toml")"
  name="$(basename "$toml" .pyproject.toml)"
  expected="$dir/expected_${name}.json"

  if [ ! -f "$expected" ]; then
    echo "skip: pm-config  $toml (no expected_${name}.json)"
    return 0
  fi

  ran=$((ran + 1))
  got="$("$bin" pm config "$toml" 2>&1)" || rc=$?
  rc="${rc:-0}"
  want="$(cat "$expected")"
  if [ "$rc" -ne 0 ]; then
    echo "FAIL: pm-config  $toml exited $rc"
    echo "stderr+stdout:"
    echo "$got"
    fail=$((fail + 1))
    return 0
  fi
  if [ "$got" != "$want" ]; then
    echo "FAIL: pm-config  $toml stdout mismatch"
    echo "  got:  $(printf '%s' "$got" | head -c 400)"
    echo "  want: $(printf '%s' "$want" | head -c 400)"
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   pm-config  $toml"
}

for toml in tests/fixtures/v01*/*.pyproject.toml; do
  [ -e "$toml" ] || continue
  run_pm_config_fixture "$toml"
done

run_pm_info_fixture() {
  local fixroot="$1"
  local pkg="$2"
  local expected="$3"
  ran=$((ran + 1))
  local got rc
  got="$(BUNPY_PYPI_FIXTURES="$fixroot" BUNPY_CACHE_DIR="$(mktemp -d)" "$bin" pm info "$pkg" 2>&1)" || rc=$?
  rc="${rc:-0}"
  local want
  want="$(cat "$expected")"
  if [ "$rc" -ne 0 ]; then
    echo "FAIL: pm-info    $pkg exited $rc"
    echo "stderr+stdout:"
    echo "$got"
    fail=$((fail + 1))
    return 0
  fi
  if [ "$got" != "$want" ]; then
    echo "FAIL: pm-info    $pkg stdout mismatch"
    echo "  got:  $(printf '%s' "$got" | head -c 400)"
    echo "  want: $(printf '%s' "$want" | head -c 400)"
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   pm-info    $pkg"
}

if [ -d tests/fixtures/v011/index ]; then
  run_pm_info_fixture tests/fixtures/v011/index widget tests/fixtures/v011/expected_widget.json
fi

echo "---"
echo "ran $ran fixtures, $fail failed"
exit "$fail"
