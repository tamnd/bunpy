---
title: Runtime
description: How bunpy executes Python - compatibility, globals, Node.js shims, and the goipy VM.
weight: 5
---

bunpy's Python runtime is built on three open-source components:

| Component | Role |
|-----------|------|
| **gopapy** | Python 3.14 parser → AST |
| **gocopy** | AST → CPython 3.14 bytecode |
| **goipy** | CPython bytecode interpreter (pure Go) |

The pipeline is fully in-process: no subprocess, no CPython installation, no
GIL. Each `bunpy run` call compiles and runs in the same OS thread as the CLI.

{{< cards >}}
  {{< card link="python" title="Python compatibility" subtitle="Which Python version, what works, what doesn't" >}}
  {{< card link="imports" title="Imports" subtitle="How bunpy resolves import statements" >}}
  {{< card link="globals" title="Injected globals" subtitle="fetch, URL, setTimeout and other web-platform globals" >}}
  {{< card link="node-compat" title="Node.js compatibility" subtitle="bunpy.node.* - the Node.js standard library shim" >}}
  {{< card link="vm" title="VM internals" subtitle="goipy bytecode interpreter architecture" >}}
{{< /cards >}}
