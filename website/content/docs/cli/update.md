---
title: bunpy update
description: Upgrade locked packages to the latest versions that satisfy constraints.
weight: 5
---

```bash
bunpy update [flags] [package...]
```

## Description

Re-resolves the dependency graph and upgrades all (or named) packages to the
newest versions that satisfy the version constraints in `pyproject.toml`.
Rewrites `bunpy.lock` with the new pins and installs the upgraded packages.

## Flags

| Flag | Description |
|------|-------------|
| `--latest <pkg>` | Strip the version constraint for `<pkg>` and upgrade to absolute latest |
| `--no-install` | Rewrite the lock only; don't install into site-packages |
| `--help`, `-h` | Print help |

## Examples

Upgrade all packages:

```bash
bunpy update
```

Upgrade only specific packages:

```bash
bunpy update requests httpx
```

Upgrade to latest regardless of constraint (strips `==` pins):

```bash
bunpy update --latest requests
```

Update the lockfile without reinstalling:

```bash
bunpy update --no-install
```

## Output

```
widget 1.0.0 -> 1.1.0
gizmo  2.1.0 -> 2.2.3
2 packages updated.
```

If no upgrades are available: `no changes.`

## Difference from install

`bunpy install` uses the existing lock to reproduce the pinned environment.
`bunpy update` re-resolves to find newer versions.
