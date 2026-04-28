---
title: bunpy version
description: Print bunpy version and build metadata.
weight: 13
---

```bash
bunpy version [flags]
bunpy --version
```

## Flags

| Flag | Description |
|------|-------------|
| `--short` | Print only the version string |
| `--json` | Print full build metadata as JSON |
| `--help`, `-h` | Print help |

## Examples

```bash
bunpy version
# bunpy 0.9.1 (linux/amd64)
# goipy  2a20bf9
# gocopy bdaac9f
# gopapy 7e162ce
# go     1.24.0

bunpy version --short
# 0.9.1

bunpy version --json
# {
#   "version": "0.9.1",
#   "commit": "abc1234",
#   "build_date": "2026-04-28T10:00:00Z",
#   "goipy": "2a20bf9",
#   "gocopy": "bdaac9f",
#   "gopapy": "7e162ce",
#   "go": "1.24.0",
#   "os": "linux",
#   "arch": "amd64"
# }
```
