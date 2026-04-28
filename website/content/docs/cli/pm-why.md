---
title: bunpy pm why
description: Show why a package is in the dependency graph.
weight: 22
---

```bash
bunpy pm why <package> [flags]
```

## Description

`bunpy pm why` traces the reverse dependency graph for a package — it answers "what pulled this in?" by walking up from the target package to the roots (your direct dependencies and the project itself). The output is a tree showing every path through which a package is required.

It reads `uv.lock` exclusively. No network access, no Python interpreter needed.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--lockfile <path>` | `uv.lock` | Lock file to inspect |
| `--json` | off | Output the tree as JSON |
| `--help`, `-h` | | Print help |

## Output

Suppose your project depends on `requests`, which depends on `certifi`. Running:

```bash
bunpy pm why certifi
```

produces:

```
certifi 2024.2.2
└── required by requests 2.31.0 (>=2017.4.17)
    └── required by httpx 0.27.0
        └── required by myproject (direct)
    └── required by myproject (direct)
```

Each line shows the package that requires the target and the version specifier it uses. When a package is reachable through multiple paths, all paths are shown. The root nodes are marked `(direct)` — these are packages you explicitly listed in `pyproject.toml`.

## Real-world example: requests → certifi

`certifi` is a common transitive dependency that projects never add directly. Here is a complete trace:

```bash
bunpy pm why certifi
```

```
certifi 2024.2.2
├── required by requests 2.31.0 (>=2017.4.17)
│   └── required by myproject (direct)
└── required by httpx 0.27.0 (*)
    └── required by myproject (direct)
```

This shows `certifi` arrives through two independent paths: `requests` and `httpx` both depend on it. If you removed both `requests` and `httpx` from `pyproject.toml`, `certifi` would be dropped from `uv.lock` automatically on the next `bunpy pm lock`.

## Another example: a deeper tree

```bash
bunpy pm why h11
```

```
h11 0.14.0
└── required by httpcore 1.0.4 (>=0.13.0,<0.15)
    └── required by httpx 0.27.0 (>=0.18.0,<0.20.0)
        └── required by myproject (direct)
```

`h11` is three levels deep. Without `pm why`, it would be opaque why it appears in `uv.lock` at all.

## JSON output

```bash
bunpy pm why certifi --json
```

```json
{
  "package": "certifi",
  "version": "2024.2.2",
  "paths": [
    [
      {"name": "certifi", "version": "2024.2.2", "specifier": null},
      {"name": "requests", "version": "2.31.0", "specifier": ">=2017.4.17"},
      {"name": "myproject", "version": null, "specifier": null, "direct": true}
    ],
    [
      {"name": "certifi", "version": "2024.2.2", "specifier": null},
      {"name": "httpx", "version": "0.27.0", "specifier": "*"},
      {"name": "myproject", "version": null, "specifier": null, "direct": true}
    ]
  ]
}
```

Each element of `paths` is an ordered list from the target package up to a root, with the specifier that was used at each step.

## Common use cases

### Deciding whether to remove a package

If you want to drop a direct dependency, `pm why` tells you whether anything else in the graph depends on it:

```bash
bunpy pm why urllib3
```

If `urllib3` shows up as required by `requests` or `httpx`, removing your direct entry in `pyproject.toml` will not drop it from `uv.lock` — it will stay as a transitive dependency.

### Auditing unexpected packages

After `bunpy install`, if you see a package in `.bunpy/site-packages/` that you do not recognise, `pm why` explains its origin immediately:

```bash
bunpy pm why sniffio
# sniffio 1.3.1
# └── required by anyio 4.3.0 (>=1.1)
#     └── required by httpx 0.27.0 (*)
#         └── required by myproject (direct)
```

### Understanding version constraint conflicts

When `bunpy pm lock` reports a conflict, `pm why` helps find who is imposing incompatible constraints on a shared transitive dependency:

```bash
bunpy pm why cryptography
```

If two direct dependencies each specify different ranges for `cryptography`, the tree will show both paths side by side, making the conflict explicit.

### Specifying a different lock file

In a monorepo with multiple lock files:

```bash
bunpy pm why certifi --lockfile services/api/uv.lock
```
