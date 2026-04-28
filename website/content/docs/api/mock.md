---
title: bunpy.mock
description: Test mocking, spying, and stubbing utilities.
---

See the [Mocking guide](/bunpy/docs/test/mocking/) for a full walkthrough.

```python
from bunpy.mock import fn, spyOn
```

## Quick reference

| Function | Description |
|----------|-------------|
| `fn(impl=None)` | Create a mock function |
| `spyOn(obj, method)` | Wrap an existing method |
| `module(name)` | Context manager to stub a module |

## Mock function API

```python
m = fn(lambda x: x * 2)
m(5)           # 10
m.mock.calls   # [[5]]
m.mock.results # [10]
m.mock.call_count  # 1

m.mockReturnValue(42)
m.mockImplementation(lambda x: x + 1)
m.mockClear()
m.mockReset()
```

## spyOn

```python
spy = spyOn(os.path, "exists")
os.path.exists("/tmp")
spy.mock.call_count  # 1
spy.mockRestore()
```
