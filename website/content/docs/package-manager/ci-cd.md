---
title: CI/CD with bunpy
description: Cache, install, test, and publish with bunpy in GitHub Actions, GitLab CI, and other pipelines.
weight: 5
---

bunpy is designed to run well in CI. Two properties make this work: the wheel cache at `~/.cache/bunpy/` is shareable across jobs, and `--frozen` refuses to modify `uv.lock` so a stale lock file is a hard error rather than a silent drift.

## Core CI principles

**Always use `--frozen`.** In CI, `bunpy install --frozen` installs exactly what is in `uv.lock` and exits with code 1 if the lock file is out of date with `pyproject.toml`. This prevents the environment from silently diverging from what developers have tested locally.

**Cache `~/.cache/bunpy/`.** Wheel downloads are cached by content-addressable hash. Restoring the cache on a cache hit drops install time from tens of seconds to under a second on warm runs.

**Commit `uv.lock`.** The lock file must be checked into version control. CI validates it; developers update it locally with `bunpy pm lock` and commit the result.

## GitHub Actions

### Full workflow

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: bunpy/setup-bunpy@v1
        with:
          bunpy-version: latest

      - name: Cache bunpy wheel store
        uses: actions/cache@v4
        with:
          path: ~/.cache/bunpy
          key: bunpy-${{ runner.os }}-${{ hashFiles('uv.lock') }}
          restore-keys: |
            bunpy-${{ runner.os }}-

      - name: Install dependencies
        run: bunpy install --frozen

      - name: Lint
        run: bunpy check

      - name: Test
        run: bunpy test
```

The cache key includes the hash of `uv.lock`. When the lock file changes (a dependency update), the key misses and bunpy re-downloads the new wheels, then writes a fresh cache. Unchanged wheels are not re-downloaded on any subsequent run.

### Matrix builds

Test across multiple Python versions:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        python-version: ["3.11", "3.12", "3.13"]

    steps:
      - uses: actions/checkout@v4

      - uses: bunpy/setup-bunpy@v1
        with:
          python-version: ${{ matrix.python-version }}

      - name: Cache bunpy wheel store
        uses: actions/cache@v4
        with:
          path: ~/.cache/bunpy
          key: bunpy-${{ runner.os }}-py${{ matrix.python-version }}-${{ hashFiles('uv.lock') }}
          restore-keys: |
            bunpy-${{ runner.os }}-py${{ matrix.python-version }}-

      - run: bunpy install --frozen
      - run: bunpy test
```

### Publishing workflow

Publish to PyPI on a version tag:

```yaml
# .github/workflows/publish.yml
name: Publish

on:
  push:
    tags:
      - "v*"

jobs:
  publish:
    runs-on: ubuntu-latest
    environment: pypi
    permissions:
      id-token: write   # for trusted publishing (OIDC)

    steps:
      - uses: actions/checkout@v4

      - uses: bunpy/setup-bunpy@v1

      - run: bunpy install --frozen

      - name: Build
        run: bunpy build --compile -o dist/myapp

      - name: Publish
        run: bunpy publish
        env:
          PYPI_TOKEN: ${{ secrets.PYPI_TOKEN }}
```

Alternatively, use PyPI's trusted publishing (OIDC) to publish without a stored secret — set `PYPI_TOKEN` to the OIDC exchange result or configure the PyPI project to accept GitHub Actions identities.

## GitLab CI

```yaml
# .gitlab-ci.yml
image: ubuntu:24.04

variables:
  BUNPY_CACHE_DIR: "$CI_PROJECT_DIR/.cache/bunpy"

cache:
  key:
    files:
      - uv.lock
  paths:
    - .cache/bunpy/

stages:
  - test
  - publish

test:
  stage: test
  before_script:
    - curl -fsSL https://bunpy.sh/install.sh | sh
    - export PATH="$HOME/.bunpy/bin:$PATH"
  script:
    - bunpy install --frozen
    - bunpy check
    - bunpy test

publish:
  stage: publish
  only:
    - tags
  script:
    - bunpy publish --token "$PYPI_TOKEN"
```

Override `BUNPY_CACHE_DIR` to a project-local path so GitLab's cache key mechanism can manage it. The cache key is tied to `uv.lock` — same semantics as the GitHub Actions setup.

## Lock file validation

`--frozen` is the install-time guard, but you can also run an explicit lock-file check to catch drift before install:

```bash
bunpy pm lock --check
```

This exits with code 1 if `uv.lock` does not match the current `pyproject.toml` without modifying either file. Add it as the first step in CI to get a clear, immediate error:

```yaml
- name: Verify lock file is up to date
  run: bunpy pm lock --check

- name: Install
  run: bunpy install --frozen
```

If the check fails, the developer needs to run `bunpy pm lock` locally and commit the updated `uv.lock`.

## Dependency auditing in CI

Add `bunpy pm audit` to catch known CVEs before they ship:

```yaml
- name: Security audit
  run: bunpy pm audit

- name: Check for outdated packages
  run: bunpy pm outdated --check
```

Run `pm audit` on every push. Run `pm outdated --check` on a weekly schedule to track version drift without blocking feature work.

## Caching strategy reference

| Cache path | What it stores | Key recommendation |
|------------|----------------|-------------------|
| `~/.cache/bunpy/` | Wheels by content hash | `hashFiles('uv.lock')` |
| `.bunpy/site-packages/` | Extracted packages | Do not cache — fast to restore from wheel cache |

Do not cache `.bunpy/site-packages/` directly. The wheel cache is the correct level: extracting wheels from the local cache is near-instant, and caching the extracted tree introduces stale-file edge cases when the lock file changes.
