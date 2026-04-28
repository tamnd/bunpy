---
title: bunpy test
description: Run Python tests with the bunpy test runner.
weight: 7
---

```bash
bunpy test [flags] [pattern...]
```

## Description

Discovers and runs test files. A file is treated as a test if its name starts
with `test_` or ends with `_test.py`. Each function decorated with `@test`
inside those files is a test case.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--filter <substr>` | | Only run tests whose name contains `substr` |
| `--verbose`, `-v` | off | Print each test name as it runs |
| `--bail`, `-b` | off | Stop after the first failure |
| `--watch`, `-w` | off | Re-run tests on file changes |
| `--timeout <ms>` | 5000 | Per-test timeout in milliseconds |
| `--coverage` | off | Emit a coverage report (experimental) |
| `--help`, `-h` | | Print help |

## Writing tests

```python
from bunpy.test import test, expect

@test("adds two numbers")
def _():
    expect(1 + 1).to_be(2)

@test("raises on bad input")
def _():
    with expect.raises(ValueError):
        int("not a number")
```

Run:

```bash
bunpy test
# ✓ adds two numbers   (0.3 ms)
# ✓ raises on bad input (0.1 ms)
# 2 passed, 0 failed
```

## Skipping tests

```python
from bunpy.test import test, expect, skip

@test("not ready yet")
def _():
    skip("WIP")
```

## Filter

Run only tests that match a substring:

```bash
bunpy test --filter "adds"
```

## Verbose mode

```bash
bunpy test --verbose
# PASS  tests/test_math.py::adds two numbers   0.3 ms
# PASS  tests/test_math.py::raises on bad input 0.1 ms
```

## Watch mode

Continuously re-runs tests on every save:

```bash
bunpy test --watch
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | All tests passed |
| 1 | One or more tests failed |
| 2 | Internal error (compile or runner failure) |
