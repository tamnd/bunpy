---
title: Python compatibility
description: Python 3.14 runtime via goipy -- what is supported, what is not, and how to check coverage.
weight: 1
---

```python
import sys
print(sys.version)
# 3.14.0 (bunpy/goipy)
```

bunpy's Python runtime is **goipy** -- a pure-Go implementation of CPython 3.14. The gocopy compiler produces CPython 3.14 bytecode; the goipy VM executes it. No system Python is required or used.

## Supported language features

Everything in the list below works as it does in CPython 3.14. If something is not in this list and not in the "not yet supported" table, assume it works -- these are just the highlights.

### Control flow

```python
# All standard control flow
for i in range(10):
    if i % 2 == 0:
        continue
    print(i)

# Pattern matching (Python 3.10+)
match command:
    case "quit":
        raise SystemExit
    case "help":
        print("Available commands: quit, help")
    case _:
        print(f"Unknown: {command}")

# Walrus operator
while chunk := file.read(8192):
    process(chunk)
```

### Functions and classes

```python
# Closures, decorators, *args, **kwargs
def retry(n):
    def decorator(fn):
        def wrapper(*args, **kwargs):
            for attempt in range(n):
                try:
                    return fn(*args, **kwargs)
                except Exception:
                    if attempt == n - 1:
                        raise
        return wrapper
    return decorator

@retry(3)
def unstable_request(url):
    return fetch(url)

# Classes with dunder methods
class Vector:
    def __init__(self, x, y):
        self.x, self.y = x, y
    def __add__(self, other):
        return Vector(self.x + other.x, self.y + other.y)
    def __repr__(self):
        return f"Vector({self.x}, {self.y})"

v = Vector(1, 2) + Vector(3, 4)
print(v)  # Vector(4, 6)
```

### Generators and async/await

```python
# Generators
def fibonacci():
    a, b = 0, 1
    while True:
        yield a
        a, b = b, a + b

gen = fibonacci()
print([next(gen) for _ in range(8)])  # [0, 1, 1, 2, 3, 5, 8, 13]

# yield from
def chain(*iterables):
    for it in iterables:
        yield from it

# Async/await
import asyncio

async def fetch_all(urls):
    results = []
    for url in urls:
        resp = fetch(url)
        results.append(resp.json())
    return results

asyncio.run(fetch_all(["https://httpbin.org/get"]))
```

### Comprehensions

```python
squares = [x**2 for x in range(10)]
even_squares = {x**2 for x in range(10) if x % 2 == 0}
word_lengths = {word: len(word) for word in ["hello", "world"]}
lazy = (x**2 for x in range(10**9))  # generator, no memory spike
```

### Type annotations

```python
# Annotations are stored but not enforced at runtime
def greet(name: str) -> str:
    return f"Hello, {name}"

from typing import Optional, list

def find(items: list[str], target: str) -> Optional[int]:
    try:
        return items.index(target)
    except ValueError:
        return None
```

### f-strings

```python
x = 3.14159
print(f"{x:.2f}")           # 3.14
print(f"{x!r}")             # 3.14159
print(f"{'hello':>10}")     # '     hello'

# Nested f-strings (Python 3.12+)
width = 10
print(f"{f'{x:.2f}':>{width}}")  # '      3.14'
```

## What is not yet supported

| Feature | Status |
|---------|--------|
| C extension modules (`.so`, `.pyd`) | Not supported. goipy is pure Go and cannot load native extensions. |
| `ctypes` | Stub only. Calls return `None`. |
| `cffi` | Not available. |
| `multiprocessing` fork mode | Goroutine-based process model. `fork` is not available; `spawn` works. |
| `__slots__` with metaclass customisation | Partial. Basic `__slots__` works; complex metaclass combinations may not. |
| Frame inspection (`sys._getframe`) | Not exposed. |
| `sys.settrace` / `sys.setprofile` | Not implemented. Profiling tools that rely on these will not work. |
| `readline` | No interactive REPL support. |
| `tkinter` and GUI frameworks | No display backend. |

If your project relies on C extensions (NumPy, Pandas, cryptography, etc.), check if a pure-Python fallback is available or use the `--backend=cpython` flag (planned for v0.11).

## Standard library coverage

bunpy ships 214 Python standard library modules. To see the full list:

```bash
bunpy stdlib
```

Key modules available:

```
asyncio    collections  concurrent  contextlib  dataclasses
datetime   enum         functools   hashlib     http.client
http.server importlib   inspect     io          itertools
json       logging      math        operator    os
os.path    pathlib      pickle      queue       random
re         shutil       signal      socket      sqlite3
ssl        string       struct      subprocess  sys
tempfile   threading    time        typing      unittest
urllib     uuid         warnings    weakref     xml
```

Modules that are stubs or partial: `ctypes`, `curses`, `dbm`, `mmap`, `readline`, `resource`, `termios`.

## Version check

```python
import sys

print(sys.version)
# 3.14.0 (bunpy/goipy)

print(sys.version_info)
# sys.version_info(major=3, minor=14, micro=0, releaselevel='final', serial=0)

print(sys.implementation.name)
# goipy
```

Use `sys.implementation.name == "goipy"` to detect the bunpy runtime in code that needs to branch:

```python
import sys

if sys.implementation.name == "goipy":
    from bunpy.fetch import fetch as http_fetch
else:
    import urllib.request
    def http_fetch(url):
        with urllib.request.urlopen(url) as r:
            return r.read()
```

## Performance

goipy executes CPython 3.14 bytecode directly. For pure-Python workloads, performance is within 1.5-3x of CPython 3.14. Workloads that depend on C extensions (NumPy, etc.) are not supported and do not apply.

Goroutine-based threading means that CPU-bound threads can run in parallel -- there is no GIL equivalent in goipy. I/O-bound code (network, file) scales well under concurrent load.
