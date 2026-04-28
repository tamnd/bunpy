---
title: CI/CD with GitHub Actions
description: Full GitHub Actions workflow for bunpy - lint, type check, test with coverage, build .pyz, and deploy. With caching and matrix builds.
---

This guide builds a production-grade GitHub Actions workflow for a bunpy project. The pipeline runs lint, type check, tests with coverage upload, builds a `.pyz` archive, and deploys on merge to main. It caches the bunpy install between runs to keep CI fast.


## Workflow overview

```
lint (ruff)
  type-check (mypy)
    test (bunpy test --coverage, matrix: 3 OS x 2 Python)
      build (.pyz archive)
        deploy (on push to main)
```

Each stage depends on the previous one. The test stage fans out across a matrix of operating systems and Python versions. The build and deploy stages run only after all matrix jobs pass.


## Full workflow file

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install bunpy
        run: curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash

      - name: Add bunpy to PATH
        run: echo "$HOME/.bunpy/bin" >> $GITHUB_PATH

      - name: Cache bunpy packages
        uses: actions/cache@v4
        with:
          path: ~/.cache/bunpy
          key: bunpy-${{ runner.os }}-${{ hashFiles('uv.lock') }}
          restore-keys: |
            bunpy-${{ runner.os }}-

      - name: Install dependencies
        run: bunpy install --frozen

      - name: Run ruff check
        run: bunpy run ruff check src/ tests/

      - name: Run ruff format check
        run: bunpy run ruff format --check src/ tests/

  type-check:
    name: Type check
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v4

      - name: Install bunpy
        run: curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash

      - name: Add bunpy to PATH
        run: echo "$HOME/.bunpy/bin" >> $GITHUB_PATH

      - name: Cache bunpy packages
        uses: actions/cache@v4
        with:
          path: ~/.cache/bunpy
          key: bunpy-${{ runner.os }}-${{ hashFiles('uv.lock') }}
          restore-keys: |
            bunpy-${{ runner.os }}-

      - name: Install dependencies
        run: bunpy install --frozen

      - name: Run mypy
        run: bunpy run mypy src/

  test:
    name: Test (${{ matrix.os }}, Python ${{ matrix.python-version }})
    runs-on: ${{ matrix.os }}
    needs: type-check
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        python-version: ["3.12", "3.14"]
    steps:
      - uses: actions/checkout@v4

      - name: Install bunpy (Unix)
        if: runner.os != 'Windows'
        run: curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash

      - name: Install bunpy (Windows)
        if: runner.os == 'Windows'
        run: |
          Invoke-WebRequest -Uri "https://tamnd.github.io/bunpy/install.ps1" -OutFile install.ps1
          ./install.ps1
        shell: pwsh

      - name: Add bunpy to PATH (Unix)
        if: runner.os != 'Windows'
        run: echo "$HOME/.bunpy/bin" >> $GITHUB_PATH

      - name: Add bunpy to PATH (Windows)
        if: runner.os == 'Windows'
        run: echo "$env:USERPROFILE\.bunpy\bin" >> $env:GITHUB_PATH
        shell: pwsh

      - name: Cache bunpy packages
        uses: actions/cache@v4
        with:
          path: |
            ~/.cache/bunpy
            ~\AppData\Local\bunpy\cache
          key: bunpy-${{ runner.os }}-py${{ matrix.python-version }}-${{ hashFiles('uv.lock') }}
          restore-keys: |
            bunpy-${{ runner.os }}-py${{ matrix.python-version }}-
            bunpy-${{ runner.os }}-

      - name: Install dependencies
        run: bunpy install --frozen

      - name: Run tests with coverage
        run: bunpy test --coverage --coverage-report=xml tests/
        env:
          DATABASE_URL: ${{ secrets.TEST_DATABASE_URL }}

      - name: Upload coverage to Codecov
        if: matrix.os == 'ubuntu-latest' && matrix.python-version == '3.14'
        uses: codecov/codecov-action@v4
        with:
          token: ${{ secrets.CODECOV_TOKEN }}
          file: ./coverage.xml
          flags: unittests
          name: bunpy-coverage
          fail_ci_if_error: false

  build:
    name: Build .pyz
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4

      - name: Install bunpy
        run: curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash

      - name: Add bunpy to PATH
        run: echo "$HOME/.bunpy/bin" >> $GITHUB_PATH

      - name: Cache bunpy packages
        uses: actions/cache@v4
        with:
          path: ~/.cache/bunpy
          key: bunpy-${{ runner.os }}-${{ hashFiles('uv.lock') }}
          restore-keys: |
            bunpy-${{ runner.os }}-

      - name: Install dependencies
        run: bunpy install --frozen

      - name: Build archive
        run: bunpy build src/myapp/__main__.py -o dist/myapp.pyz

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: myapp-pyz
          path: dist/myapp.pyz
          retention-days: 7

  deploy:
    name: Deploy
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    environment: production
    steps:
      - uses: actions/checkout@v4

      - name: Download artifact
        uses: actions/download-artifact@v4
        with:
          name: myapp-pyz
          path: dist/

      - name: Deploy to Fly.io
        uses: superfly/flyctl-actions/setup-flyctl@master

      - name: Run deploy
        run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```


## Cache key strategy

The cache key uses the hash of `uv.lock`:

```yaml
key: bunpy-${{ runner.os }}-${{ hashFiles('uv.lock') }}
```

When `uv.lock` changes (a dependency was added, removed, or updated), the hash changes and the cache is busted. The fresh packages are installed and saved as the new cache entry. When `uv.lock` does not change, the cache hit skips `bunpy install` entirely.

The `restore-keys` fallback:

```yaml
restore-keys: |
  bunpy-${{ runner.os }}-
```

This allows a partial cache restore when the exact key does not exist. Partial restores reduce install time even when the lockfile has changed, because only new packages need to be downloaded.


## Matrix builds

The `fail-fast: false` setting allows all matrix combinations to run to completion even if one fails. This is useful for finding platform-specific failures: a test that fails on Windows should not hide a separate failure on macOS.

To run the matrix only on pull requests and skip it on direct pushes to reduce cost:

```yaml
strategy:
  matrix:
    os: ${{ github.event_name == 'pull_request' && fromJSON('["ubuntu-latest", "macos-latest", "windows-latest"]') || fromJSON('["ubuntu-latest"]') }}
    python-version: ["3.12", "3.14"]
```


## Secrets management

Set secrets in the GitHub repository under Settings > Secrets and variables > Actions:

| Secret | Purpose |
|---|---|
| `CODECOV_TOKEN` | Upload coverage reports to Codecov |
| `FLY_API_TOKEN` | Authenticate flyctl for deployment |
| `TEST_DATABASE_URL` | Database connection string for tests |

Secrets are injected as environment variables at runtime. They never appear in logs (GitHub Actions redacts them automatically).

For environment-specific secrets (staging vs production), use GitHub Environments. The `deploy` job above references the `production` environment, which can have its own secret set and require manual approval before the job runs.


## Codecov integration

The coverage upload step only runs on `ubuntu-latest` with Python 3.14 to avoid uploading duplicate reports from the matrix:

```yaml
if: matrix.os == 'ubuntu-latest' && matrix.python-version == '3.14'
```

Add a `codecov.yml` to the repository root to configure coverage targets:

```yaml
coverage:
  status:
    project:
      default:
        target: 80%
        threshold: 1%
    patch:
      default:
        target: 70%
```

This fails the coverage check if overall coverage drops below 80%, or if the lines changed in the PR are covered below 70%.


## Concurrency control

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

This cancels any in-progress run for the same workflow and branch when a new commit is pushed. For pull requests, each new push cancels the previous CI run, saving runner minutes.


## Branch protection

To enforce the workflow before merging:

1. Go to Settings > Branches > Add rule
2. Branch name pattern: `main`
3. Enable "Require status checks to pass before merging"
4. Add `Lint`, `Type check`, `Test (ubuntu-latest, Python 3.14)`, and `Build .pyz` as required checks

With this in place, no one can merge a PR that breaks lint, type checks, tests, or the build.
