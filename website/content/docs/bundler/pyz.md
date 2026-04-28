---
title: .pyz format
description: The portable Python ZIP archive format used by bunpy build.
weight: 1
---

## What is a .pyz?

A `.pyz` file is a standard Python ZIP application - a ZIP archive that Python
(and bunpy) can execute directly. It contains:

```
myapp.pyz
├── __main__.py      # entry point
├── myapp/           # your package
│   ├── __init__.py
│   └── server.py
└── requests/        # vendored dependencies
    └── ...
```

The archive has a `#!/usr/bin/env bunpy` shebang prepended so it is directly
executable on POSIX systems.

## Building a .pyz

```bash
bunpy build app.py
# → app.pyz  (24 KB)
```

Specify the output name:

```bash
bunpy build -o dist/myapp.pyz src/myapp/__main__.py
```

## Running a .pyz

```bash
bunpy app.pyz
# or, after chmod +x:
./app.pyz
```

## What gets bundled

bunpy traces `import` statements statically and includes:

- Your entry point and all reachable local modules
- Any wheels installed in `.bunpy/site-packages/` that are imported

Standard library modules are **not** bundled - they are provided by the goipy
VM at run time.

## Size

A minimal script bundles to a few KB. A script importing `requests` typically
bundles to ~200 KB.

## .pyz vs --compile

| | `.pyz` | `--compile` |
|-|--------|-------------|
| Requires bunpy | Yes | No |
| File size | Small | Larger (embeds VM) |
| Cross-compile | N/A | Yes |
| Startup time | ~5 ms | ~2 ms |
