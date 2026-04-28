---
title: bunpy run
description: Execute a Python script with the bunpy runtime.
weight: 1
---

```bash
bunpy run [flags] <script.py> [-- script-args...]
bunpy <script.py> [script-args...]   # shorthand
```

## Description

Compiles a Python source file with gocopy and runs it on the goipy VM.
Injected globals (`fetch`, `URL`, `Request`, `Response`, `WebSocket`,
`setTimeout`, `setInterval`, `clearTimeout`, `clearInterval`) are available
without any import.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--inspect` | off | Print IR size, module list, and timing breakdown, then exit |
| `--quiet`, `-q` | off | Suppress runtime warnings |
| `--env-file <path>` | `.env` | Load environment variables from a file before running |
| `--watch`, `-w` | off | Re-run the script when source files change |
| `--help`, `-h` | | Print help |

## Examples

Run a script:

```bash
bunpy run hello.py
bunpy hello.py       # identical
```

Pass arguments to the script (available as `sys.argv`):

```bash
bunpy run cli.py -- --port 8080 --debug
```

Inspect the compiled IR:

```bash
bunpy run --inspect hello.py
# IR size:    4.2 KB
# Compile:    1.3 ms
# Marshal:    0.1 ms
# Unmarshal:  0.2 ms
# Run:        0.8 ms
```

Load a `.env` file:

```bash
bunpy run --env-file .env.production server.py
```

Watch mode — re-runs on every save:

```bash
bunpy run --watch dev.py
```

## Script arguments

Arguments after `--` are forwarded to the script and available via `sys.argv`:

```python
# cli.py
import sys
print(sys.argv)
```

```bash
bunpy cli.py -- foo bar
# ['cli.py', 'foo', 'bar']
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `BUNPY_NO_COLOR` | Disable ANSI color output |
| `BUNPY_MAX_DEPTH` | Override the VM recursion limit (default 500) |
