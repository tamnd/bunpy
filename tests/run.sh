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

run_install_wheel_fixture() {
  local whl="$1"
  local dir name expected got rc want target
  dir="$(dirname "$whl")"
  name="$(basename "$whl" .whl)"
  expected="$dir/expected_${name}.txt"

  if [ ! -f "$expected" ]; then
    echo "skip: install-wh $whl (no expected_${name}.txt)"
    return 0
  fi

  ran=$((ran + 1))
  target="$(mktemp -d)"
  got=""
  rc=0
  if ! "$bin" pm install-wheel "$whl" --target "$target" >/dev/null 2>&1; then
    rc=$?
    echo "FAIL: install-wh $whl install exited $rc"
    fail=$((fail + 1))
    return 0
  fi
  got="$(cd "$target" && find . -type f | LC_ALL=C sort)"
  want="$(cat "$expected")"
  if [ "$got" != "$want" ]; then
    echo "FAIL: install-wh $whl tree mismatch"
    echo "  got:"
    printf '%s\n' "$got" | sed 's/^/    /'
    echo "  want:"
    printf '%s\n' "$want" | sed 's/^/    /'
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   install-wh $whl"
}

for whl in tests/fixtures/v01*/*.whl; do
  [ -e "$whl" ] || continue
  run_install_wheel_fixture "$whl"
done

run_add_fixture() {
  local in="$1"
  local dir name input expected_pyproject expected_files args work cache rc=0
  dir="$(dirname "$in")"
  name="$(basename "$in" .add_in)"
  input="$dir/${name}.input.toml"
  expected_pyproject="$dir/expected_${name}.toml"
  expected_files="$dir/expected_${name}_files.txt"

  if [ ! -f "$input" ] || [ ! -f "$expected_pyproject" ] || [ ! -f "$expected_files" ]; then
    echo "skip: add        $in (missing inputs/expectations)"
    return 0
  fi

  ran=$((ran + 1))
  args="$(cat "$in")"
  work="$(mktemp -d)"
  cache="$(mktemp -d)"
  cp "$input" "$work/pyproject.toml"

  if ! ( cd "$work" && BUNPY_PYPI_FIXTURES="$ROOT/$dir/index" BUNPY_CACHE_DIR="$cache" "$bin" add $args >/dev/null 2>&1 ); then
    rc=$?
    echo "FAIL: add        $in exited $rc"
    fail=$((fail + 1))
    return 0
  fi

  local got_pyproject want_pyproject got_files want_files
  got_pyproject="$(cat "$work/pyproject.toml")"
  want_pyproject="$(cat "$expected_pyproject")"
  if [ "$got_pyproject" != "$want_pyproject" ]; then
    echo "FAIL: add        $in pyproject mismatch"
    echo "  got:"
    printf '%s\n' "$got_pyproject" | sed 's/^/    /'
    echo "  want:"
    printf '%s\n' "$want_pyproject" | sed 's/^/    /'
    fail=$((fail + 1))
    return 0
  fi

  got_files="$(cd "$work/.bunpy/site-packages" && find . -type f | LC_ALL=C sort)"
  want_files="$(cat "$expected_files")"
  if [ "$got_files" != "$want_files" ]; then
    echo "FAIL: add        $in files mismatch"
    echo "  got:"
    printf '%s\n' "$got_files" | sed 's/^/    /'
    echo "  want:"
    printf '%s\n' "$want_files" | sed 's/^/    /'
    fail=$((fail + 1))
    return 0
  fi

  echo "ok:   add        $in"
}

for in in tests/fixtures/v01*/*.add_in; do
  [ -e "$in" ] || continue
  run_add_fixture "$in"
done

run_lock_fixture() {
  local in="$1"
  local dir name input expected args work cache rc=0 fixroot
  dir="$(dirname "$in")"
  name="$(basename "$in" .lock_in)"
  input="$dir/${name}.input.toml"
  expected="$dir/expected_${name}.lock"

  if [ ! -f "$input" ] || [ ! -f "$expected" ]; then
    echo "skip: lock       $in (missing inputs/expectations)"
    return 0
  fi

  fixroot="$ROOT/$dir/index"
  if [ ! -d "$fixroot" ]; then
    fixroot="$ROOT/tests/fixtures/v013/index"
  fi

  ran=$((ran + 1))
  args="$(cat "$in")"
  work="$(mktemp -d)"
  cache="$(mktemp -d)"
  cp "$input" "$work/pyproject.toml"
  # Optional seed lockfile: lets fixtures exercise verbs (update,
  # outdated) that read an existing bunpy.lock instead of writing a
  # fresh one from scratch.
  if [ -f "$dir/${name}.seed.lock" ]; then
    cp "$dir/${name}.seed.lock" "$work/bunpy.lock"
  fi

  if ! ( cd "$work" && BUNPY_PYPI_FIXTURES="$fixroot" BUNPY_CACHE_DIR="$cache" "$bin" $args >/dev/null 2>&1 ); then
    rc=$?
    echo "FAIL: lock       $in exited $rc"
    fail=$((fail + 1))
    return 0
  fi

  local got want
  got="$(grep -v '^generated = ' "$work/bunpy.lock")"
  want="$(cat "$expected")"
  if [ "$got" != "$want" ]; then
    echo "FAIL: lock       $in lockfile mismatch"
    echo "  got:"
    printf '%s\n' "$got" | sed 's/^/    /'
    echo "  want:"
    printf '%s\n' "$want" | sed 's/^/    /'
    fail=$((fail + 1))
    return 0
  fi
  echo "ok:   lock       $in"
}

for in in tests/fixtures/v01*/*.lock_in; do
  [ -e "$in" ] || continue
  run_lock_fixture "$in"
done

echo "---"
echo "ran $ran fixtures, $fail failed"
exit "$fail"
