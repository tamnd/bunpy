---
title: bunpy check
description: Run static analysis and lint checks on Python source.
weight: 9
---

```bash
bunpy check [flags] [path...]
```

## Description

Runs the built-in linter and checker over Python source files. Reports
undefined names, unused imports, type mismatches (where inferable), and
style violations.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--select <rule,...>` | all | Enable only the listed rules |
| `--ignore <rule,...>` | | Disable the listed rules |
| `--fix` | off | Auto-fix fixable violations in place |
| `--format <fmt>` | text | Output format: `text`, `json` |
| `--help`, `-h` | | Print help |

## Examples

Check all files:

```bash
bunpy check
```

Check a directory:

```bash
bunpy check src/
```

Auto-fix what can be fixed:

```bash
bunpy check --fix src/
```

CI: emit machine-readable JSON:

```bash
bunpy check --format json > lint.json
```
