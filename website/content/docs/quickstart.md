---
title: Quickstart
description: Run your first Python script, install packages, start a server, and write tests -- all with bunpy in under five minutes.
weight: 2
---

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
export PATH="$HOME/.bunpy/bin:$PATH"
bunpy --version
# bunpy 0.10.29 (linux/amd64)
```

## Run a script

Create a file and run it directly:

```python
# hello.py
print("Hello from bunpy!")
```

```bash
bunpy hello.py
# Hello from bunpy!
```

No compile step, no virtualenv, no interpreter to configure. bunpy embeds Python 3.14 (via goipy) and runs the file immediately.

## Use the standard library

All 214 goipy-provided stdlib modules are available out of the box:

```python
# stdlib.py
import json
import math
import datetime

data = {
    "pi": math.pi,
    "e": math.e,
    "now": datetime.datetime.utcnow().isoformat(),
}
print(json.dumps(data, indent=2))
```

```bash
bunpy stdlib.py
# {
#   "pi": 3.141592653589793,
#   "e": 2.718281828459045,
#   "now": "2026-04-28T10:00:00.000000"
# }
```

## Use the fetch global

`fetch` is injected into every script's global scope. No import needed:

```python
# fetch_example.py
resp = fetch("https://httpbin.org/get")
data = resp.json()
print(data["url"])
print(data["headers"]["User-Agent"])
```

```bash
bunpy fetch_example.py
# https://httpbin.org/get
# bunpy/0.10.29
```

For POST requests:

```python
resp = fetch("https://httpbin.org/post", {
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": '{"name": "bunpy"}',
})
print(resp.status)  # 200
```

## Start a project

Use `bunpy init` to scaffold a new project with `pyproject.toml`:

```bash
bunpy init myapp
cd myapp
```

The generated `pyproject.toml`:

```toml
[project]
name = "myapp"
version = "0.1.0"
requires-python = ">=3.14"
dependencies = []
```

## Install packages

Add a package from PyPI:

```bash
bunpy add requests httpx
```

This updates `pyproject.toml`, resolves the full dependency graph, downloads the wheels, and writes `uv.lock`:

```toml
# pyproject.toml after bunpy add requests httpx
[project]
dependencies = [
    "requests>=2.31.0",
    "httpx>=0.27.0",
]
```

Then use the packages in your script:

```python
# main.py
import requests

r = requests.get("https://httpbin.org/get")
print(r.status_code)   # 200
print(r.json()["url"]) # https://httpbin.org/get
```

```bash
bunpy run main.py
# 200
# https://httpbin.org/get
```

## Start an HTTP server

```python
# server.py
from bunpy.serve import serve

def handler(req):
    if req.path == "/health":
        return {"status": 200, "body": "ok"}
    name = req.query.get("name", "world")
    return {"status": 200, "body": f"Hello, {name}!"}

print("Listening on http://localhost:3000")
serve(handler, port=3000)
```

```bash
bunpy server.py
# Listening on http://localhost:3000
```

In another terminal:

```bash
curl http://localhost:3000?name=bunpy
# Hello, bunpy!
curl http://localhost:3000/health
# ok
```

## Write and run tests

```python
# tests/test_math.py
from bunpy.test import test, expect

@test("addition is commutative")
def _():
    expect(1 + 2).to_be(3)
    expect(2 + 1).to_be(3)

@test("division by zero raises")
def _():
    def bad():
        return 1 / 0
    expect(bad).to_raise(ZeroDivisionError)
```

```bash
bunpy test
# bunpy test v0.10.29 (myapp)
#
# tests/test_math.py:
# ✓ addition is commutative [0.12ms]
# ✓ division by zero raises [0.08ms]
#
# 2 tests passed (2)
```

## Bundle to a .pyz

Package your project as a standalone `.pyz` archive that runs with `python3` or with `bunpy`:

```bash
bunpy build
# Built myapp.pyz (1.2 MB)
```

```bash
bunpy run myapp.pyz
# or
python3 myapp.pyz
```

The `.pyz` contains your source, all dependencies, and a `__main__.py` entry point. It does not embed the goipy runtime, so the target machine needs either bunpy or a CPython 3.14 install.

## The lockfile

Every `bunpy add` and `bunpy install` updates `uv.lock`. Commit this file to version control. It guarantees that `bunpy install` produces byte-for-byte identical installs on every machine:

```bash
git add pyproject.toml uv.lock
git commit -m "add requests and httpx"
```

## Next steps

- [Installation](/docs/installation/) -- platform-specific install options
- [CLI reference](/docs/cli/) -- all bunpy commands
- [Runtime docs](/docs/runtime/) -- Python compatibility, globals, import resolution
- [API reference](/docs/api/) -- all `bunpy.*` modules
- [Package manager](/docs/package-manager/) -- lockfile, workspaces, patches
