---
title: Project templates
description: Scaffold new projects with bunpy create.
weight: 3
---

`bunpy create` scaffolds a new project from a built-in template. No network
access needed — templates are embedded in the bunpy binary.

## Usage

```bash
bunpy create <template> <project-name> [--yes]
```

`--yes` skips interactive prompts and accepts all defaults.

## Available templates

```bash
bunpy create --list
```

| Template | Description |
|----------|-------------|
| `app` | Application with `__main__.py` entry point |
| `lib` | Library package with `__init__.py` |
| `script` | Single-file standalone script |
| `workspace` | Monorepo workspace with `[tool.bunpy.workspace]` |

## app

A runnable application with a `src/` layout:

```bash
bunpy create app myapp --yes
```

Generated structure:

```
myapp/
├── pyproject.toml
├── README.md
└── src/
    └── myapp/
        ├── __init__.py
        └── __main__.py
```

`pyproject.toml`:

```toml
[project]
name = "myapp"
version = "0.1.0"
requires-python = ">=3.12"
```

`__main__.py`:

```python
def main():
    print("Hello from myapp!")

if __name__ == "__main__":
    main()
```

Run:

```bash
cd myapp
bunpy src/myapp/__main__.py
```

## lib

A reusable library with an `__init__.py` and test scaffold:

```bash
bunpy create lib mylib --yes
```

Generated structure:

```
mylib/
├── pyproject.toml
├── README.md
└── src/
    └── mylib/
        └── __init__.py
```

## script

A single-file script — minimal boilerplate, just a `.py` file:

```bash
bunpy create script myscript --yes
```

Generated structure:

```
myscript/
└── myscript.py
```

`myscript.py`:

```python
#!/usr/bin/env bunpy
"""myscript — a standalone bunpy script."""


def main():
    print("Hello!")


if __name__ == "__main__":
    main()
```

Run directly:

```bash
bunpy myscript.py
```

Or make it executable:

```bash
chmod +x myscript.py
./myscript.py
```

## workspace

A monorepo root with two starter member packages:

```bash
bunpy create workspace myws --yes
```

Generated structure:

```
myws/
├── pyproject.toml
└── pkgs/
    ├── alpha/
    │   └── pyproject.toml
    └── beta/
        └── pyproject.toml
```

Root `pyproject.toml`:

```toml
[project]
name = "myws"
version = "0.1.0"

[tool.bunpy.workspace]
members = ["pkgs/alpha", "pkgs/beta"]
```

Inspect the workspace:

```bash
cd myws
bunpy workspace --list
```

## Interactive mode

Without `--yes`, bunpy prompts for project name, version, and description:

```
$ bunpy create app
Project name: myapp
Version (0.1.0):
Description: My new bunpy app
✓ Created myapp/
```
