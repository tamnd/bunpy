---
title: "bunpy: an all-in-one Python toolkit"
description: "Runtime, package manager, bundler, test runner, and formatter in one fast static binary. No virtualenv, no system Python, no setup."
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
  An all-in-one Python toolkit. Runtime, package manager, bundler, test
  runner, and formatter in one fast static binary.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-button text="Get started" link="/bunpy/docs/installation/" >}}
{{< hextra/hero-button text="GitHub" link="https://github.com/tamnd/bunpy" style="secondary" >}}
</div>

<div class="bp-install">

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
bunpy --version
# bunpy 0.11.15
```

</div>

<p class="hx-text-sm hx-text-gray-500 hx-mt-2 hx-mb-20">
macOS (arm64, x64) · Linux (arm64, x64) · Windows (x64)
</p>

{{< hextra/feature-grid cols=3 >}}
  {{< hextra/feature-card
    title="Designed for speed"
    subtitle="Written in Go. Cold starts in milliseconds, package resolution in seconds, no Python subprocess fan-out."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[200px]"
  >}}
  {{< hextra/feature-card
    title="All in one"
    subtitle="Replaces pip, pytest, ruff, and black. A single ~4 MB static binary. Copy it anywhere and it works."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[200px]"
  >}}
  {{< hextra/feature-card
    title="Zero config"
    subtitle="Drop a pyproject.toml and run bunpy install. No virtualenv, no requirements.txt, no activation step."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[200px]"
  >}}
{{< /hextra/feature-grid >}}

<div class="hx-mt-20 hx-mb-4">

## What's inside

</div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Package manager"
    subtitle="Install, lock, and update PyPI packages. Reads `pyproject.toml`. Workspaces, overrides, audits, patches."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="HTTP server"
    subtitle="`bunpy.serve` handles routing, parsing, and response serialisation in Go. No framework required."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Bundler"
    subtitle="Bundle to a portable `.pyz` archive or compile to a self-contained native binary."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Test runner"
    subtitle="Run tests with `bunpy test`. Built-in coverage, reporters, filtering, and lifecycle hooks."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Web globals"
    subtitle="`fetch`, `URL`, `Request`, `Response`, and `WebSocket` are available in every script. No import needed."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Native modules"
    subtitle="SQL, Redis, S3, JWT, crypto, YAML, CSV, cron, and password hashing ship with the binary."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
{{< /hextra/feature-grid >}}

<div class="hx-mt-20 hx-mb-4">

## Compared to a Python toolchain

</div>

<div class="bp-compare">

|                          | bunpy                  | python + pip + venv     |
|--------------------------|------------------------|-------------------------|
| Runtime included         | yes (Python 3.14)      | requires CPython        |
| Package install          | built-in `bunpy add`   | pip + manual venv       |
| Test runner              | built-in `bunpy test`  | pytest (separate)       |
| Bundler                  | built-in `bunpy build` | shiv / pyinstaller      |
| Formatter                | built-in `bunpy fmt`   | black / ruff (separate) |
| HTTP server              | built-in `bunpy.serve` | flask / fastapi         |
| Native binary executable | `bunpy build --compile`| not supported           |

</div>

<div class="hx-mt-20 hx-mb-4">

## Quick look

</div>

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
# fetch is available globally, no import needed
resp = fetch("https://api.github.com/repos/tamnd/bunpy")
print(resp.json()["stargazers_count"])
```

```bash
# Package manager
bunpy add requests fastapi
bunpy install --frozen
bunpy pm lock
```

<div class="hx-mt-20 hx-mb-4">

## Commands

</div>

| Command | What it does |
|---------|-------------|
| `bunpy script.py` | Run a Python script |
| `bunpy add requests` | Add a dependency |
| `bunpy install` | Install from `pyproject.toml` |
| `bunpy pm lock` | Resolve and pin all dependencies |
| `bunpy build app.py` | Bundle to `app.pyz` |
| `bunpy test` | Run tests |
| `bunpy fmt src/` | Format source |
| `bunpy repl` | Interactive REPL |
