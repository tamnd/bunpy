---
title: Python compatibility
description: Which Python version bunpy targets and what is supported.
weight: 1
---

## Target version

bunpy targets **Python 3.14** syntax and semantics. The gocopy compiler
produces CPython 3.14 bytecode; the goipy VM implements the CPython 3.14
instruction set.

## What works

- All standard control flow: `if`, `for`, `while`, `with`, `try/except/finally`
- Functions, closures, decorators, `*args`, `**kwargs`
- Classes, inheritance, `super()`, dunder methods
- Generators and `yield`, `yield from`
- Async/await (`async def`, `await`, `async for`, `async with`)
- Comprehensions: list, dict, set, generator
- Type annotations (evaluated lazily; no runtime enforcement)
- f-strings (including nested), format strings
- Pattern matching (`match` / `case`, Python 3.10+)
- Walrus operator (`:=`)
- Most of the Python standard library (214 modules exposed by goipy)

## What is not yet supported

| Feature | Status |
|---------|--------|
| C extension modules (`.so`, `.pyd`) | Not supported — goipy is pure Go |
| `ctypes` | Stub only |
| `multiprocessing` fork mode | Goroutine-based; `fork` not available |
| `__slots__` with metaclass customisation | Partial |
| Frame inspection (`sys._getframe`) | Not exposed |
| `sys.settrace` / `sys.setprofile` | Not implemented |

## Stdlib coverage

bunpy ships 214 Python standard library modules (as of v0.9.1). Run:

```bash
bunpy stdlib
```

to list all available modules.

## Version check

```python
import sys
print(sys.version)
# 3.14.0 (bunpy/goipy)
```
