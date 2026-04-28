---
title: bunpy build
description: Bundle a Python script to a portable .pyz archive or native binary.
weight: 6
---

```bash
bunpy build [flags] <script.py>
```

## Description

Compiles a Python script and its imports into a portable `.pyz` archive.
With `--compile`, cross-compiles to a standalone native binary that does not
require bunpy to run.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--outfile`, `-o` | `<script>.pyz` | Output file path |
| `--compile` | off | Compile to a native binary (embeds the goipy VM) |
| `--target <os-arch>` | host | Cross-compile target: `linux-x64`, `linux-arm64`, `darwin-x64`, `darwin-arm64`, `windows-x64` |
| `--minify` | off | Minify Python source before bundling |
| `--define <K=V>` | | Replace `K` with the literal `V` in source at compile time |
| `--sourcemap` | off | Emit a `.pyz.map` source map alongside the bundle |
| `--watch` | off | Rebuild on file changes |
| `--help`, `-h` | | Print help |

## Examples

Bundle to `.pyz`:

```bash
bunpy build app.py
# → app.pyz
```

Specify output name:

```bash
bunpy build -o dist/myapp.pyz app.py
```

Run the bundle:

```bash
bunpy app.pyz
# or, if bunpy is on PATH:
./app.pyz
```

Compile to a native binary for the current host:

```bash
bunpy build --compile app.py -o dist/myapp
./dist/myapp
```

Cross-compile for Linux x64 from macOS:

```bash
bunpy build --compile --target linux-x64 app.py -o dist/myapp-linux
```

Strip dead branches at compile time with `--define`:

```bash
bunpy build --define DEBUG=False app.py
```

Watch mode during development:

```bash
bunpy build --watch app.py
```

## .pyz format

A `.pyz` file is a ZIP archive with a `__main__.py` entry point and a
`#!` shebang pointing to bunpy. It is runnable with `bunpy <file>.pyz` or,
if marked executable, directly as `./myapp.pyz`.
