---
title: Test runner
description: Write and run tests with bunpy test.
weight: 7
---

bunpy has a built-in test runner. No third-party framework required.

```bash
bunpy test
```

{{< cards >}}
  {{< card link="mocking" title="Mocking" subtitle="Replace functions and modules with test doubles" >}}
  {{< card link="snapshots" title="Snapshots" subtitle="Assert complex values with stored snapshots" >}}
  {{< card link="watch" title="Watch mode" subtitle="Re-run tests automatically on file changes" >}}
{{< /cards >}}

## Quick start

```python
# tests/test_math.py
from bunpy.test import test, expect

@test("addition")
def _():
    expect(1 + 1).to_be(2)

@test("string contains")
def _():
    expect("hello world").to_contain("world")
```

```bash
bunpy test
# ✓ addition       (0.1 ms)
# ✓ string contains (0.1 ms)
# 2 passed, 0 failed
```

## Test discovery

bunpy finds test files by convention:

- Files named `test_*.py`
- Files named `*_test.py`

All `@test` decorated functions inside those files are collected and run.

## expect API

| Assertion | Description |
|-----------|-------------|
| `.to_be(val)` | Strict equality (`==`) |
| `.to_equal(val)` | Deep equality |
| `.to_be_none()` | Value is `None` |
| `.to_be_truthy()` | Truthy value |
| `.to_be_falsy()` | Falsy value |
| `.to_contain(val)` | String or list containment |
| `.to_have_length(n)` | `len(val) == n` |
| `.to_raise(exc)` | Callable raises exception |
| `.to_be_greater_than(n)` | `val > n` |
| `.to_be_less_than(n)` | `val < n` |
| `.to_be_close_to(n, delta)` | Floating point proximity |

### Negation

```python
expect(1 + 1).not_.to_be(3)
```

### Async tests

```python
from bunpy.test import test, expect
import asyncio

@test("async fetch")
async def _():
    resp = await asyncio.coroutine(fetch)("https://httpbin.org/get")
    expect(resp.status_code).to_be(200)
```

## Lifecycle hooks

```python
from bunpy.test import test, expect, beforeAll, afterAll, beforeEach, afterEach

@beforeAll
def setup():
    # runs once before all tests in this file
    pass

@afterEach
def cleanup():
    # runs after each test
    pass
```
