---
title: bunpy
layout: hextra-home
toc: false
---

{{< hextra/hero-badge >}}
  <span>v0.11.15 out now</span>
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  Python at Go speed
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Run, install, bundle, and test Python with one binary.
  No CPython. No pip. No config.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-button text="Get started" link="/bunpy/docs/installation/" >}}
{{< hextra/hero-button text="GitHub" link="https://github.com/tamnd/bunpy" style="secondary" >}}
</div>

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
bunpy --version  # bunpy 0.11.15
```

<p class="hx-text-sm hx-text-gray-500 hx-mt-2">
macOS (arm64, x64) · Linux (arm64, x64) · Windows (x64)
</p>


## Numbers

<div class="hx-grid hx-grid-cols-1 md:hx-grid-cols-3 hx-gap-6 hx-mt-6 hx-mb-8">
<div>

**16x faster**

`bunpy pm lock` resolves and pins packages 16 times faster than uv on the same dependency graph. Written in Go, no Python subprocess.

</div>
<div>

**One binary**

A single ~4 MB static executable replaces pip, uv, pytest, ruff, and black. Copy it anywhere and it works.

</div>
<div>

**Zero config**

Drop a `pyproject.toml` and run `bunpy install`. No virtualenv, no `requirements.txt`, no activation step.

</div>
</div>


## Tools

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Package manager"
    subtitle="Add, remove, lock, and install packages. Compatible with uv.lock. 16x faster resolution than uv."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="HTTP server"
    subtitle="Built-in bunpy.serve handles routing, parsing, and response serialisation in Go. No framework required."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Bundler"
    subtitle="Bundle to a portable .pyz archive or compile to a self-contained native binary with bunpy build --compile."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Test runner"
    subtitle="Run tests with bunpy test. Built-in coverage, reporters, filtering, and lifecycle hooks."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Web globals"
    subtitle="fetch, URL, Request, Response, and WebSocket are injected into every script. No import needed."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Native modules"
    subtitle="SQL, Redis, S3, JWT, crypto, YAML, CSV, cron, and password hashing ship with the binary."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
{{< /hextra/feature-grid >}}


## Quick look

```python
# server.py
from bunpy.serve import serve

def handler(req):
    name = req.query.get("name", "world")
    return f"Hello, {name}!"

serve(handler, port=3000)
```

```bash
bunpy server.py
# Listening on http://localhost:3000
```

```python
# fetch is a global, no import needed
resp = fetch("https://api.github.com/repos/tamnd/bunpy")
print(resp.json()["stargazers_count"])
```

```bash
# Package manager
bunpy add requests fastapi
bunpy install --frozen   # CI-safe, fails if uv.lock is stale
bunpy pm lock            # 16x faster than uv
```


## Commands

| Command | What it does |
|---------|-------------|
| `bunpy script.py` | Run a Python script |
| `bunpy add requests` | Add a dependency |
| `bunpy install` | Install from pyproject.toml |
| `bunpy pm lock` | Resolve and pin all dependencies |
| `bunpy build app.py` | Bundle to app.pyz |
| `bunpy test` | Run tests |
| `bunpy fmt src/` | Format source |
| `bunpy repl` | Interactive REPL |
