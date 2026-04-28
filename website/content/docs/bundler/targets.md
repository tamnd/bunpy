---
title: Build targets
description: Cross-compile bunpy binaries for different operating systems and architectures.
weight: 5
---

```bash
bunpy build --compile --target <os>-<arch> app.py -o dist/myapp
```

`--target` is only available with `--compile`. It cross-compiles the embedded
goipy VM for the specified platform.

## Supported targets

| Target | Platform |
|--------|----------|
| `linux-x64` | Linux, amd64 |
| `linux-arm64` | Linux, ARM64 |
| `darwin-x64` | macOS, Intel |
| `darwin-arm64` | macOS, Apple Silicon |
| `windows-x64` | Windows, amd64 |
| `windows-arm64` | Windows, ARM64 |

## Examples

Build for Linux from macOS:

```bash
bunpy build --compile --target linux-x64 app.py -o dist/myapp-linux
```

Build for Windows:

```bash
bunpy build --compile --target windows-x64 app.py -o dist/myapp.exe
```

Build all targets in a loop:

```bash
for target in linux-x64 linux-arm64 darwin-x64 darwin-arm64 windows-x64; do
  ext=""
  [[ "$target" == windows* ]] && ext=".exe"
  bunpy build --compile --target "$target" app.py -o "dist/myapp-${target}${ext}"
done
```

## File sizes

A "hello world" compiled binary is approximately:

| Target | Size |
|--------|------|
| linux-x64 | ~8 MB |
| darwin-arm64 | ~8 MB |
| windows-x64 | ~9 MB |

Size includes the embedded goipy VM, gocopy compiler, and the Python script.
