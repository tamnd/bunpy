---
title: Filtering tests
description: Run a subset of tests by pattern, file, or modifier.
weight: 7
---

Running the full suite on every change is slow. bunpy provides several ways to narrow which tests run: pattern matching, inline `.only` and `.skip` modifiers, file selection, and flags that control failure behavior and retry logic.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--grep <pattern>` | none | Run only tests whose name matches the pattern |
| `--bail` | off | Stop after the first test failure |
| `--timeout <ms>` | `5000` | Per-test timeout in milliseconds |
| `--retry <n>` | `0` | Retry each failing test up to `n` times |
| `--help`, `-h` | | Print help |

## Running a single file

Pass a file path as an argument to run only the tests in that file:

```bash
bunpy test tests/test_auth.py
```

Pass multiple files:

```bash
bunpy test tests/test_auth.py tests/test_user.py
```

Use a glob:

```bash
bunpy test tests/api/**
```

## --grep

`--grep` filters tests by name using a case-sensitive substring match. Any test whose name contains the pattern runs; all others are skipped.

```bash
bunpy test --grep "login"
```

```
✓ auth › login with valid credentials (2.1 ms)
✓ auth › login with invalid password (0.9 ms)
✓ auth › login with expired token (3.4 ms)
3 passed, 0 failed
```

Use a regular expression by wrapping the pattern in slashes:

```bash
bunpy test --grep "/^auth/"
```

This matches any test name starting with `auth`. Regex syntax follows Python's `re` module.

Combine with a file path to narrow further:

```bash
bunpy test tests/test_auth.py --grep "expired"
```

```
✓ auth › login with expired token (3.4 ms)
1 passed, 0 failed
```

## .only

Mark a test with `.only` to run just that test when the file is collected. All other tests in the file are skipped.

```python
from bunpy.test import test, expect

@test("login works")
def _():
    expect(1 + 1).to_be(2)

@test.only("registration works")
def _():
    # only this test runs
    expect(True).to_be_truthy()

@test("logout works")
def _():
    # this test is skipped
    expect(False).to_be_truthy()
```

```bash
bunpy test tests/test_auth.py
```

```
✓ registration works (0.4 ms)
1 passed, 0 failed, 2 skipped
```

`.only` is a development tool. It should not be committed - leave it in and you silently skip the rest of the suite. bunpy will warn if `.only` is found when `CI=true` is set.

## .skip

Mark a test with `.skip` to exclude it without deleting it. The test is counted as skipped in the summary.

```python
@test.skip("flaky network test")
def _():
    # not run
    resp = requests.get("https://example.com")
    expect(resp.status_code).to_be(200)
```

```
- flaky network test (skipped)
1 skipped
```

Use `.skip` to suppress a known-broken test while tracking the work to fix it, rather than commenting it out and forgetting about it.

## --bail

Stop the entire test run after the first failure. Useful when a fundamental setup error causes every subsequent test to fail - stopping early gives you a cleaner signal.

```bash
bunpy test --bail
```

```
✓ auth › login with valid credentials (2.1 ms)
✗ auth › login with invalid password (0.9 ms)
  AssertionError: expected 401, got 500

Bailed after 1 failure. 1 passed, 1 failed.
```

Remaining tests are not run and not counted.

## --timeout

Each test has a default timeout of 5000 ms. If a test takes longer, it fails with a timeout error:

```
✗ slow integration test (5001 ms)
  Error: test timed out after 5000 ms
```

Raise the timeout for tests that legitimately take longer:

```bash
bunpy test --timeout 30000
```

Or lower it to enforce snappiness in a unit test suite:

```bash
bunpy test --timeout 500
```

The timeout clock starts when the test function is called and stops when it returns (or its `async` coroutine resolves). Setup hooks (`beforeAll`, `beforeEach`) run outside the per-test timeout.

## --retry

Retry failing tests automatically, up to a specified number of times. The test passes if any attempt succeeds.

```bash
bunpy test --retry 3
```

This is useful for tests that interact with real network services or external processes that occasionally flake. It is not a substitute for fixing flaky tests - but it buys time while the underlying issue is diagnosed.

```
✗ payment gateway responds (attempt 1/3)
✗ payment gateway responds (attempt 2/3)
✓ payment gateway responds (attempt 3/3) (1821 ms)
1 passed (retried 2 times)
```

Set `--retry` per-test in `pyproject.toml` for finer control:

```toml
[tool.bunpy.test]
retry = 2          # global default
timeout = 10000    # global default (ms)
```

## Combining flags

All flags compose:

```bash
# Run only auth tests, stop on first failure, retry up to 2 times
bunpy test tests/test_auth.py --grep "login" --bail --retry 2

# Tight timeout for the unit suite
bunpy test tests/unit/ --timeout 200 --bail

# Run a single file in watch mode with grep
bunpy test tests/test_billing.py --watch --grep "charge"
```

## Summary of filter precedence

When multiple filters are active, they apply in order:

1. File paths - only the specified files are collected.
2. `--grep` - only tests matching the pattern are collected from those files.
3. `.only` - if any `.only` exists in a collected file, all non-`.only` tests in that file are skipped.
4. `.skip` - matching tests are removed from the collected set.
5. `--bail` - stops execution after the first failure in the remaining set.
