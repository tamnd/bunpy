---
title: Mocking
description: Replace functions and modules with test doubles using bunpy.mock.
weight: 1
---

`bunpy.mock` provides `mock`, `spy`, and `stub` utilities for replacing
implementations during tests.

## mock.fn

Create a mock function that records calls:

```python
from bunpy.mock import fn

add = fn(lambda a, b: a + b)

result = add(1, 2)

assert add.mock.calls == [[1, 2]]
assert add.mock.results == [3]
assert add.mock.call_count == 1
```

## mock.spyOn

Wrap an existing method and record calls without changing behaviour:

```python
from bunpy.mock import spyOn
import os

spy = spyOn(os.path, "exists")

os.path.exists("/tmp")

assert spy.mock.call_count == 1
spy.mockRestore()   # restore the original
```

## mock.module

Replace an entire module with a stub for the duration of a test:

```python
from bunpy.mock import module as mockModule
from bunpy.test import test, expect

@test("calls external API")
def _():
    with mockModule("requests") as m:
        m.get.return_value.status_code = 200
        import myapp.client
        resp = myapp.client.fetch_data()
        expect(resp["status"]).to_be(200)
```

## mockReturnValue

Override the return value without changing the call recording:

```python
get = fn()
get.mockReturnValue(42)

assert get() == 42
assert get.mock.call_count == 1
```

## mockImplementation

Swap the implementation dynamically:

```python
greet = fn(lambda name: f"Hello, {name}!")
greet.mockImplementation(lambda name: f"Hi, {name}!")

assert greet("Alice") == "Hi, Alice!"
```

## Clearing mocks

```python
add.mockClear()   # resets call count and history
add.mockReset()   # also resets return value and implementation
```
