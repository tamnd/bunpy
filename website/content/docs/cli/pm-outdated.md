---
title: bunpy pm outdated
description: List packages that have newer versions available on PyPI.
weight: 21
---

```bash
bunpy pm outdated [flags]
```

## Description

`bunpy pm outdated` compares the versions pinned in `uv.lock` against the latest releases on PyPI and lists every package where a newer version is available. It reads the lock file, not the virtualenv — so it works even before `bunpy install` has run.

Run it periodically to keep dependencies fresh, or in CI to enforce a freshness policy.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--check` | off | Exit with code 1 if any package is outdated (no output change) |
| `--lockfile <path>` | `uv.lock` | Path to the lock file to inspect |
| `--json` | off | Output results as JSON |
| `--help`, `-h` | | Print help |

## Output

```bash
bunpy pm outdated
```

```
Package              Installed    Latest     Type
------------------------------------------------------------
certifi              2023.7.22    2024.2.2   patch
charset-normalizer   3.3.0        3.3.2      patch
httpx                0.24.1       0.27.0     minor
pydantic             1.10.13      2.6.4      major
requests             2.31.0       2.32.2     minor
urllib3              2.0.7        2.2.1      patch
```

The `Type` column classifies the version gap by semver component:

- `patch` — only the patch version changed; generally safe to update
- `minor` — new features added; read the changelog
- `major` — breaking changes possible; plan the upgrade

## JSON output

```bash
bunpy pm outdated --json
```

```json
[
  {
    "package": "httpx",
    "installed": "0.24.1",
    "latest": "0.27.0",
    "type": "minor"
  },
  {
    "package": "pydantic",
    "installed": "1.10.13",
    "latest": "2.6.4",
    "type": "major"
  }
]
```

JSON output is sorted alphabetically by package name and writes to stdout, so it is safe to pipe.

## Updating outdated packages

`pm outdated` is read-only — it never modifies `uv.lock` or `pyproject.toml`. To actually update packages, use:

```bash
# Update a single package
bunpy update httpx

# Update all packages to their latest allowed versions
bunpy update

# Check what will change before committing
bunpy update --dry-run
```

`bunpy update` respects the version constraints in `pyproject.toml`. If `httpx = ">=0.24,<0.25"` is pinned, `pm outdated` will still report the gap, but `bunpy update` will not advance past the constraint.

## CI integration

### Fail the build on any outdated package

Use `--check` to get a non-zero exit code when outdated packages are found:

```bash
bunpy pm outdated --check
```

This is useful as a scheduled CI job that keeps the team aware of drift:

```yaml
# .github/workflows/outdated.yml
name: Dependency freshness

on:
  schedule:
    - cron: "0 9 * * 1"   # every Monday at 09:00 UTC

jobs:
  outdated:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: bunpy/setup-bunpy@v1
      - run: bunpy install --frozen
      - run: bunpy pm outdated --check
```

The job fails if any package has a newer release. The team then decides which upgrades to apply.

### Report only, never fail

Omit `--check` if you want visibility without blocking builds:

```bash
bunpy pm outdated || true
```

Or capture the JSON and post it to a Slack webhook, a GitHub issue, or a dashboard.

### Combining with audit

Run both checks together in one CI step:

```bash
bunpy pm outdated --check && bunpy pm audit
```

`pm outdated` catches packages with new releases (which may contain security fixes), and `pm audit` catches packages with known CVEs regardless of version age.

## Interpreting major version gaps

A `major` bump (like `pydantic 1.x → 2.x`) almost always means a breaking API change. Do not run `bunpy update pydantic` blindly. Instead:

1. Read the migration guide for the package.
2. Pin the upgrade in a feature branch.
3. Run `bunpy test` to surface failures.
4. Resolve deprecations before merging.

`pm outdated` surfaces the gap early so you can schedule the work, rather than discovering it during an emergency patch.
