---
title: bunpy pm info
description: Show metadata for a package from PyPI.
weight: 20
---

```bash
bunpy pm info <package> [flags]
```

## Description

`bunpy pm info` fetches and displays metadata for any package on PyPI — installed or not. Use it to inspect a dependency before adding it, verify a version's license, or check what it depends on without installing anything.

The data comes directly from the PyPI JSON API. No virtualenv or lock file is needed.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | off | Output raw JSON instead of the formatted display |
| `--version <ver>` | latest | Show metadata for a specific version |
| `--help`, `-h` | | Print help |

## Output

```bash
bunpy pm info requests
```

```
name:        requests
version:     2.31.0
summary:     Python HTTP for Humans.
homepage:    https://requests.readthedocs.io
author:      Kenneth Reitz
license:     Apache 2.0
requires-python: >=3.7

dependencies:
  certifi >=2017.4.17
  charset-normalizer >=2,<4
  idna >=2.5,<4
  urllib3 >=1.21.1,<3

classifiers:
  Development Status :: 5 - Production/Stable
  Intended Audience :: Developers
  License :: OSI Approved :: Apache Software License
  Programming Language :: Python :: 3
  Topic :: Internet :: WWW/HTTP
```

Each field maps directly to the corresponding PyPI metadata field. The `dependencies` list shows `Requires-Dist` entries from the wheel metadata, which are the same constraints that go into `uv.lock` when you run `bunpy add`.

## JSON output

Pass `--json` to get the full PyPI payload as structured JSON. Useful in scripts or when piping into `jq`:

```bash
bunpy pm info requests --json
```

```json
{
  "name": "requests",
  "version": "2.31.0",
  "summary": "Python HTTP for Humans.",
  "home_page": "https://requests.readthedocs.io",
  "author": "Kenneth Reitz",
  "license": "Apache 2.0",
  "requires_python": ">=3.7",
  "requires_dist": [
    "certifi>=2017.4.17",
    "charset-normalizer>=2,<4",
    "idna>=2.5,<4",
    "urllib3>=1.21.1,<3"
  ],
  "classifiers": [
    "Development Status :: 5 - Production/Stable",
    "Intended Audience :: Developers",
    "License :: OSI Approved :: Apache Software License",
    "Programming Language :: Python :: 3",
    "Topic :: Internet :: WWW/HTTP"
  ],
  "project_urls": {
    "Documentation": "https://requests.readthedocs.io",
    "Source": "https://github.com/psf/requests"
  }
}
```

Pipe into `jq` to extract a single field:

```bash
bunpy pm info requests --json | jq '.license'
# "Apache 2.0"

bunpy pm info requests --json | jq '.requires_dist[]'
# "certifi>=2017.4.17"
# "charset-normalizer>=2,<4"
# "idna>=2.5,<4"
# "urllib3>=1.21.1,<3"
```

## Inspecting a specific version

Check what changed in an older release without touching your environment:

```bash
bunpy pm info requests --version 2.28.0
```

This is useful when upgrading: compare the dependency list between your installed version and the latest to understand what else might shift.

## Checking before you add

A common workflow is to inspect a package before committing it to `pyproject.toml`:

```bash
# 1. Check the license
bunpy pm info httpx --json | jq '.license'
# "BSD License"

# 2. Check what it pulls in
bunpy pm info httpx --json | jq '.requires_dist[]'
# "anyio"
# "certifi"
# "h11>=0.13,<0.15"
# "httpcore>=0.18.0,<0.19.0"
# "sniffio"

# 3. Add if satisfied
bunpy add httpx
```

This prevents surprises: you know the transitive footprint before the package lands in `uv.lock`.

## Checking an installed package

`bunpy pm info` works on any PyPI package, installed or not. For packages already in your project, the version shown reflects the latest on PyPI, not the pinned version in your lock file. To see what version is actually installed, use:

```bash
bunpy pm outdated          # compare installed vs latest
bunpy pm why <package>     # see why it is in your graph
```

## Errors

If the package does not exist on PyPI, bunpy exits with a non-zero code and prints:

```
error: package "badpkg" not found on PyPI
```

If the network is unavailable, it exits with:

```
error: could not reach PyPI (https://pypi.org/pypi/requests/json): connection refused
```

No fallback to a local cache is attempted — `pm info` is intentionally a live query.
