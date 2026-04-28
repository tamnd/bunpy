---
title: VM internals
description: How the goipy bytecode interpreter works.
weight: 5
---

## Pipeline overview

```
source.py
  → gopapy (parser)  → AST
  → gocopy (compiler) → CPython 3.14 bytecode
  → marshal hop       → goipy object.Code
  → goipy (VM)        → result
```

Each stage is a separate Go module wired together by bunpy's `runtime`
package.

## gopapy — the parser

gopapy is a pure-Go Python 3.14 parser. It accepts UTF-8 source and produces
a concrete syntax tree that gocopy walks to emit bytecode. gopapy supports:

- Full Python 3.14 grammar including pattern matching, `type` statements,
  PEP 695 generics, and f-string nesting
- Error recovery for better IDE-style diagnostics

## gocopy — the compiler

gocopy translates the gopapy AST into CPython 3.14 bytecode (`.pyc` format).
It handles:

- Scope analysis (locals, closures, globals, builtins)
- Constant folding
- Comprehension code objects
- Class and function code objects with nested closures

## goipy — the interpreter

goipy is a pure-Go CPython 3.14 bytecode interpreter. It implements the
CPython frame-based execution model:

- Each function call creates a `Frame` with its own locals and evaluation stack
- Exceptions propagate as `*object.Exception` Go errors with full tracebacks
- Generators are implemented as Go coroutines (goroutines with channels)
- `asyncio` coroutines run on goroutines; `await` is a channel receive

## Threading model

bunpy runs Python code on Go goroutines. `threading.Thread` maps 1:1 to a
goroutine. There is no GIL — goroutines can run concurrently, but individual
Python `object.*` operations are not goroutine-safe by default. Shared state
must be protected by `threading.Lock()`.

## Recursion limit

The default recursion limit is 500 frames (configurable via
`BUNPY_MAX_DEPTH`). Exceeding it raises `RecursionError`.

## Stack frames and tracebacks

Exceptions carry a `*object.Traceback` linked list. `goipyVM.FormatException`
formats it in CPython's style:

```
Traceback (most recent call last):
  File "hello.py", line 3, in foo
    return bar()
  File "hello.py", line 7, in bar
    raise ValueError("oops")
ValueError: oops
```
