---
title: bunpy create
description: Scaffold a new project from a built-in template.
weight: 11
---

```bash
bunpy create [flags] <template> [project-name]
bunpy create --list
```

See [Project templates](/bunpy/docs/templates/) for a full walkthrough of each
template's generated structure.

## Flags

| Flag | Description |
|------|-------------|
| `--yes`, `-y` | Accept all defaults without interactive prompts |
| `--list` | Print available template names and exit |
| `--help`, `-h` | Print help |

## Available templates

| Template | Description |
|----------|-------------|
| `app` | Runnable application with `__main__.py` |
| `lib` | Library package with `__init__.py` |
| `script` | Single-file standalone script |
| `workspace` | Monorepo workspace |

## Examples

```bash
bunpy create app myapp --yes
bunpy create lib mylib
bunpy create script myscript --yes
bunpy create workspace myws --yes
bunpy create --list
```
