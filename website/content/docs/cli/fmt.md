---
title: bunpy fmt
description: Format Python source files.
weight: 8
---

```bash
bunpy fmt [flags] [path...]
```

## Description

Formats Python source files in place. Without a path argument, formats all
`.py` files reachable from the current directory. The formatter follows PEP 8
with a configurable line length.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--check` | off | Exit non-zero if any file would be reformatted (CI mode) |
| `--line-length <n>` | 88 | Maximum line length |
| `--diff` | off | Print unified diff instead of writing files |
| `--help`, `-h` | | Print help |

## Examples

Format all files in the current directory:

```bash
bunpy fmt
```

Format a specific file or directory:

```bash
bunpy fmt src/
bunpy fmt src/myapp/main.py
```

CI check - fails if formatting would change any file:

```bash
bunpy fmt --check
```

Print diff without writing:

```bash
bunpy fmt --diff src/
```
