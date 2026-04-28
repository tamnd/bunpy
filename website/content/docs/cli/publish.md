---
title: bunpy publish
description: Publish a package to PyPI.
weight: 12
---

```bash
bunpy publish [flags]
```

## Description

Builds a wheel from `pyproject.toml` and uploads it to PyPI (or a compatible
index). Requires a PyPI API token.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--token <tok>` | `$PYPI_TOKEN` | PyPI API token |
| `--index <url>` | PyPI | Upload URL |
| `--dry-run` | off | Build the wheel but don't upload |
| `--help`, `-h` | | Print help |

## Examples

Publish using an environment variable:

```bash
export PYPI_TOKEN=pypi-...
bunpy publish
```

Dry run to check the wheel:

```bash
bunpy publish --dry-run
```

Publish to a private index:

```bash
bunpy publish --index https://pypi.mycompany.com/legacy/ --token $PRIVATE_TOKEN
```

## Errors

If no token is provided and `$PYPI_TOKEN` is not set, bunpy exits with an
error. Use `--dry-run` to test the build without credentials.
