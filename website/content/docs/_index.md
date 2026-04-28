---
title: Documentation
description: bunpy is an all-in-one toolkit for Python development. Runtime, package manager, bundler, test runner, and formatter in one static binary.
weight: 1
next: /docs/installation
---

bunpy is an all-in-one toolkit for developing Python applications. It ships as a single executable called `bunpy`.

{{< hextra/feature-grid cols=2 >}}
  {{< hextra/feature-card
    title="Runtime"
    subtitle="Execute Python scripts directly. No virtualenv, no system Python required. `bunpy script.py`"
    link="runtime/"
  >}}
  {{< hextra/feature-card
    title="Package manager"
    subtitle="Install packages from PyPI, lock dependencies, manage workspaces. `bunpy add requests`"
    link="package-manager/"
  >}}
  {{< hextra/feature-card
    title="Test runner"
    subtitle="Built-in test runner with coverage, watch mode, and parallel execution. `bunpy test`"
    link="test/"
  >}}
  {{< hextra/feature-card
    title="Bundler"
    subtitle="Bundle to a portable .pyz archive or compile to a self-contained binary. `bunpy build`"
    link="bundler/"
  >}}
{{< /hextra/feature-grid >}}

New to bunpy? Start here:

{{< cards >}}
  {{< card link="installation" title="Install bunpy" icon="download" subtitle="Supported platforms and all install methods" >}}
  {{< card link="quickstart" title="Quickstart" icon="play" subtitle="Hello world in under five minutes" >}}
{{< /cards >}}

---

## What is bunpy?

bunpy is an all-in-one toolkit for Python apps. It ships as a single binary with no external dependencies.

At its core is an embedded Python 3.14 runtime powered by [goipy](https://github.com/tamnd/goipy) — a pure-Go implementation of the CPython interpreter. It starts in milliseconds, requires no system Python install, and runs scripts exactly as CPython would.

```bash
bunpy script.py          # run a Python script
bunpy add requests        # install a package from PyPI
bunpy install             # install from pyproject.toml
bunpy pm lock             # resolve and pin all dependencies
bunpy build app.py        # bundle to app.pyz
bunpy test                # run tests in tests/
bunpy fmt src/            # format source files
bunpy repl                # interactive REPL
```

The `bunpy` binary also includes a package manager, test runner, bundler, and formatter — all significantly faster than the equivalent standalone tools.

---

## What is a Python runtime?

Python is a language specification. Any piece of software that reads a Python source file and executes it is a Python *runtime*. The canonical implementation is CPython, written in C. Others include PyPy (JIT-compiled Python), MicroPython (embedded systems), and Jython (JVM-hosted).

### CPython

CPython compiles Python source to bytecode and interprets that bytecode in a virtual machine written in C. It is the reference implementation — if a program behaves a certain way in CPython, that is by definition what Python means.

### goipy

goipy is a Go implementation of the CPython 3.14 semantics. It compiles Python source to the same bytecode CPython produces, then executes that bytecode in a virtual machine written in Go. The result is a single, statically-linked binary with no dependency on a C runtime or a system Python installation.

goipy is not a subset. It aims for full CPython 3.14 compatibility. As of v0.10.29, 214 of 263 stdlib modules pass the full CPython test suite. The remaining 49 are either in progress or permanently out of scope (GUI modules such as `tkinter`).

bunpy embeds goipy. When you run `bunpy script.py`, goipy executes the file.

---

## Design goals

bunpy is designed from the ground up for the typical Python developer workflow.

**Speed.** Cold script starts in under 10 ms on an M-series Mac. Package resolution for a 47-package tree in 85 ms on a warm cache. No subprocess fan-out — the runtime, resolver, and test runner all run in the same process.

**All in one.** One binary replaces pip, pytest, black/ruff, and shiv. No virtualenv to activate, no `pip install` before every project. Drop a `pyproject.toml` and run `bunpy install`.

**CPython compatibility.** bunpy is not a new language or a simplified subset. Code that runs on CPython 3.14 should run on bunpy without changes. When there is an incompatibility, it is a bug.

**Zero config.** `pyproject.toml` is the only project file. No `setup.cfg`, no `setup.py`, no `MANIFEST.in`. The test runner discovers `tests/test_*.py` automatically. The bundler uses your `[project]` table without extra configuration.

**Portable output.** `bunpy build` produces a `.pyz` archive that runs on any machine with CPython 3.12+. `bunpy build --compile` produces a self-contained native binary that requires nothing at all.
