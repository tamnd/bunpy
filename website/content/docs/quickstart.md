---
title: Quickstart
description: Run your first Python script with bunpy in under 60 seconds.
weight: 2
---

## Install bunpy

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
```

Add bunpy to your PATH (or restart your terminal):

```bash
export PATH="$HOME/.bunpy/bin:$PATH"
```

Verify:

```bash
bunpy --version
# bunpy 0.9.1 (linux/amd64)
```

## Write your first script

```bash
mkdir hello-bunpy && cd hello-bunpy
```

Create `hello.py`:

```python
print("Hello from bunpy!")
```

Run it:

```bash
bunpy hello.py
# Hello from bunpy!
```

## Use the standard library

```python
import json
import math

data = {"pi": math.pi, "e": math.e}
print(json.dumps(data, indent=2))
```

```bash
bunpy hello.py
# {
#   "pi": 3.141592653589793,
#   "e": 2.718281828459045
# }
```

## Fetch data from the web

bunpy injects `fetch` as a global — no import required:

```python
resp = fetch("https://httpbin.org/get")
data = resp.json()
print(data["url"])
```

```bash
bunpy hello.py
# https://httpbin.org/get
```

## Start a project with dependencies

```bash
bunpy init myapp
cd myapp
```

This creates `pyproject.toml`:

```toml
[project]
name = "myapp"
version = "0.1.0"
```

Add a dependency:

```bash
bunpy add requests
```

Then use it:

```python
import requests

r = requests.get("https://httpbin.org/get")
print(r.status_code)
```

```bash
bunpy run src/myapp/__main__.py
# 200
```

## Run tests

```python
# tests/test_hello.py
from bunpy.test import test, expect

@test("addition works")
def _():
    expect(1 + 1).to_be(2)
```

```bash
bunpy test
# ✓ addition works
```

## Next steps

- [Installation](/bunpy/docs/installation/) — platform-specific install options
- [CLI reference](/bunpy/docs/cli/) — all bunpy commands
- [Runtime docs](/bunpy/docs/runtime/) — Python compatibility, globals, Node.js shims
- [API reference](/bunpy/docs/api/) — all bunpy.* modules
