---
title: Compile to native binary
description: Ship a self-contained executable that does not require bunpy.
weight: 7
---

```bash
bunpy build --compile app.py -o dist/myapp
```

`--compile` embeds the goipy VM, the gocopy compiler, and the bundled Python
source into a single native binary. The binary runs on a machine that has never
had bunpy installed.

## How it works

1. `bunpy build` compiles the script to a `.pyz` in memory.
2. It wraps the `.pyz` inside a Go binary that contains:
   - The goipy bytecode interpreter
   - The gocopy compiler (for hot-reload and sub-process support)
   - All bunpy built-in modules
   - The `.pyz` archive as an embedded resource
3. The Go linker produces a single statically-linked binary.

The result behaves identically to running `bunpy run app.py` but without the
bunpy CLI.

## Size

The compiled binary includes the full VM (~8 MB on amd64). The Python source
adds only a few KB. Total is typically 8–12 MB.

## Cross-compilation

See [Build targets](/bunpy/docs/bundler/targets/) for cross-compile examples.

## Limitations

- Native C extension modules (`.so`, `.pyd`) cannot be embedded
- The binary includes all stdlib modules; unused ones cannot be stripped yet

## Comparison

| | `bunpy run` | `.pyz` | `--compile` |
|-|-------------|--------|-------------|
| Requires bunpy | Yes | Yes | No |
| Single file | No | Yes | Yes |
| Cross-compile | N/A | N/A | Yes |
| Startup | ~5 ms | ~5 ms | ~2 ms |
