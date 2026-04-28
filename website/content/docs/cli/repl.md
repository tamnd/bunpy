---
title: bunpy repl
description: Interactive Python REPL powered by the bunpy runtime.
weight: 10
---

```bash
bunpy repl [flags]
```

## Description

Opens an interactive Python REPL that uses the same goipy VM as `bunpy run`.
All injected globals (`fetch`, `URL`, `setTimeout`, etc.) and `bunpy.*`
modules are available without import.

## Flags

| Flag | Description |
|------|-------------|
| `--quiet`, `-q` | Suppress the welcome banner |
| `--help`, `-h` | Print help |

## REPL commands

| Command | Description |
|---------|-------------|
| `:quit` / `:q` | Exit the REPL |
| `:help` | Show REPL help |

## Examples

Start the REPL:

```bash
bunpy repl
# bunpy repl 0.9.1 (type :quit to exit)
# >>>
```

Evaluate expressions:

```
>>> 1 + 1
2
>>> import math
>>> math.pi
3.141592653589793
```

Use injected globals:

```
>>> resp = fetch("https://httpbin.org/get")
>>> resp.status_code
200
```

Pipe input (non-interactive mode — used in tests):

```bash
printf 'x = 1 + 1\nprint(x)\n:quit\n' | bunpy repl --quiet
# 2
```
