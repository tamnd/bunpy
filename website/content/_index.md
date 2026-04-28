---
title: bunpy
layout: hextra-home
toc: false
---

{{< hextra/hero-badge >}}
  <span>v0.8.9 — now available</span>
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  The Python runtime,&nbsp;package manager, and bundler&nbsp;—&nbsp;all&nbsp;in&nbsp;one
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  bunpy runs Python at Go speed. One binary. Zero config.
  Comes with a package manager, bundler, test runner, linter,
  and a Node.js-compatible stdlib — out of the box.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-button text="Get started" link="/bunpy/docs/installation/" >}}
{{< hextra/hero-button text="GitHub" link="https://github.com/tamnd/bunpy" style="secondary" >}}
</div>

```bash
# macOS / Linux
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash

bunpy --version  # → bunpy 0.8.9
```

<p class="hx-text-sm hx-text-gray-500 hx-mt-2">
Works on macOS (arm64, x64) · Linux (arm64, x64) · Windows (x64)
</p>

---

## Why bunpy?

<div class="hx-grid hx-grid-cols-1 md:hx-grid-cols-3 hx-gap-6 hx-mt-6 hx-mb-8">
<div>

**Fast**

Pure Go interpreter — no CPython, no GIL, no subprocess overhead.
Goroutine-backed async. Near-Go throughput for I/O-heavy workloads.

</div>
<div>

**Complete**

Everything Bun has for JavaScript, but for Python: package manager,
bundler, test runner, formatter, linter, REPL, and a full Node.js
API compatibility shim.

</div>
<div>

**Portable**

Bundle to `.pyz` — one file that runs anywhere bunpy is installed —
or compile to a native binary with `bunpy build --compile`.

</div>
</div>

---

## Features

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="One binary"
    subtitle="Runtime + package manager + bundler + test runner + linter + formatter, shipped as a single static Go binary. Nothing to install beyond bunpy itself."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Node.js-compatible APIs"
    subtitle="Import bunpy.node.fs, bunpy.node.http, bunpy.node.crypto, bunpy.node.stream and more — same API shapes as the Node.js standard library."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Bundle to .pyz"
    subtitle="Ship your app as a portable self-contained .pyz archive with a shebang. Runs anywhere bunpy is installed, or compile to a native binary."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Async without boilerplate"
    subtitle="bunpy.asyncio.run(), gather(), create_task() — goroutine-backed concurrency with no event loop ceremony or import overhead."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Web-first globals"
    subtitle="fetch, URL, Request, Response, WebSocket injected as globals — no import statement needed. Write HTTP clients exactly like in a browser."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
  {{< hextra/feature-card
    title="Batteries included"
    subtitle="SQL, Redis, S3, JWT, crypto, YAML, CSV, templates, email, cron, password hashing — all native modules, all zero additional dependencies."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
  >}}
{{< /hextra/feature-grid >}}

---

## Node.js compatibility shim

{{< callout type="info" >}}
**bunpy 0.8+** ships `bunpy.node.*` — a full Node.js standard library shim
backed by Go stdlib. Import `bunpy.node.fs`, `bunpy.node.path`,
`bunpy.node.crypto`, `bunpy.node.http`, `bunpy.node.stream`,
`bunpy.node.zlib`, and `bunpy.node.worker_threads` with the same API shapes
as Node.js.

[Read the Node.js compatibility docs →](/bunpy/docs/node/)
{{< /callout >}}

---

## What's in the box

| Command | What it does |
|---------|-------------|
| `bunpy run script.py` | Run a Python script |
| `bunpy install` | Install dependencies from pyproject.toml |
| `bunpy add requests` | Add a package |
| `bunpy build app.py` | Bundle to app.pyz |
| `bunpy test` | Run all tests |
| `bunpy fmt src/` | Format Python source |
| `bunpy check src/` | Static lint check |
| `bunpy repl` | Interactive REPL |
