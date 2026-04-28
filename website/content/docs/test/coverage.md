---
title: Code coverage
description: Measure and enforce test coverage with bunpy test --coverage.
weight: 4
---

```bash
bunpy test --coverage [flags]
```

## Description

`bunpy test --coverage` instruments your source code as tests run and reports which lines, branches, and functions were executed. Coverage data is collected by the goipy VM's built-in tracer - no third-party tool like `coverage.py` is required.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--coverage` | off | Enable coverage collection |
| `--coverage-reporter <fmt>` | `text` | `text`, `html`, `lcov`, or `json` |
| `--coverage-dir <path>` | `coverage/` | Directory for HTML and lcov output |
| `--coverage-threshold <n>` | none | Fail if total line coverage is below `n` percent |
| `--coverage-exclude <glob>` | | Glob patterns for files to exclude |
| `--help`, `-h` | | Print help |

## Text output

```bash
bunpy test --coverage
```

```
✓ auth.login (2.1 ms)
✓ auth.logout (0.8 ms)
✓ user.create (3.4 ms)
✓ user.delete (1.2 ms)
4 passed, 0 failed (18 ms)

Coverage report:
--------------------------------------------------------------------
File                        Stmts   Miss  Branch  BrMiss  Cover
--------------------------------------------------------------------
src/myapp/__init__.py           4      0       0       0   100%
src/myapp/auth.py              42      3       8       1    91%
src/myapp/models.py            81      6      14       4    88%
src/myapp/utils.py             23     14       6       5    35%
src/myapp/cli.py               55     55      10      10     0%
--------------------------------------------------------------------
TOTAL                         205     78      38      20    73%
--------------------------------------------------------------------
```

Lines with no test coverage are shown in the `Miss` column. `Branch` and `BrMiss` track conditional branches - a line with `if/else` counts as two branches; both must be exercised for full coverage.

## HTML output

```bash
bunpy test --coverage --coverage-reporter html
```

Writes an interactive HTML report to `coverage/index.html`. Open it in a browser to see line-by-line highlighting: green lines were executed, red lines were not. Click any file to drill into the source.

```bash
open coverage/index.html
```

## lcov output

```bash
bunpy test --coverage --coverage-reporter lcov
```

Writes `coverage/lcov.info` in the [lcov trace file format](https://man7.org/linux/man-pages/man1/lcov.1.html). This is the format used by most CI coverage integrations, including Codecov, Coveralls, and SonarQube.

## JSON output

```bash
bunpy test --coverage --coverage-reporter json
```

Writes `coverage/coverage.json`:

```json
{
  "total": {
    "lines": {"total": 205, "covered": 127, "pct": 61.95},
    "branches": {"total": 38, "covered": 18, "pct": 47.37}
  },
  "files": {
    "src/myapp/auth.py": {
      "lines": {"total": 42, "covered": 39, "pct": 92.86},
      "branches": {"total": 8, "covered": 7, "pct": 87.5},
      "uncovered_lines": [34, 35, 36]
    }
  }
}
```

## Coverage thresholds

Fail the test run if coverage drops below a threshold:

```bash
bunpy test --coverage --coverage-threshold 80
```

If total line coverage is below 80%, the command exits with code 1:

```
Coverage: 73% - below threshold 80%
```

This makes coverage a hard requirement in CI.

### Per-file thresholds via pyproject.toml

Configure thresholds and other coverage options in `pyproject.toml`:

```toml
[tool.bunpy.coverage]
threshold = 80
reporters = ["text", "lcov"]
dir = "coverage"
exclude = [
  "tests/**",
  "src/myapp/migrations/**",
  "**/conftest.py",
]
```

Options in `pyproject.toml` are the defaults; CLI flags override them.

## Excluding files

Exclude files that should not count toward coverage - migrations, generated code, CLI entry points that are exercised manually:

```bash
bunpy test --coverage --coverage-exclude "src/myapp/migrations/**" --coverage-exclude "src/myapp/cli.py"
```

Or in `pyproject.toml`:

```toml
[tool.bunpy.coverage]
exclude = [
  "src/myapp/migrations/**",
  "src/myapp/cli.py",
  "**/__main__.py",
]
```

Excluded files do not appear in the report and do not count toward the threshold.

## CI integration with Codecov

```yaml
# .github/workflows/ci.yml
- name: Test with coverage
  run: bunpy test --coverage --coverage-reporter lcov --coverage-threshold 80

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    files: coverage/lcov.info
    fail_ci_if_error: true
```

Codecov reads `coverage/lcov.info`, posts a comment on the pull request showing coverage diff, and tracks coverage trends over time. The `--coverage-threshold 80` flag ensures the build fails before the upload step if coverage drops too far.

## Viewing uncovered lines

The text reporter shows which lines were missed:

```
src/myapp/utils.py             23     14       6       5    35%
```

To see the exact lines, use the HTML reporter, or add `--verbose` to the text reporter:

```bash
bunpy test --coverage --verbose
```

```
src/myapp/utils.py
  Not covered: lines 12-18, 23, 31-36, 45
```

Write tests for those lines, or exclude the file if the code is unreachable in tests by design (e.g., a `__main__` guard).
