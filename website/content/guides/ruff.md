---
title: Linting and formatting with ruff
description: Add ruff to a bunpy project — pyproject.toml config, ruff check and format, pre-commit hook, VS Code extension, and CI integration.
---

ruff is a fast Python linter and formatter written in Rust. It replaces flake8, isort, pyupgrade, and black with a single tool that runs in milliseconds. This guide covers adding ruff to a bunpy project, configuring it for FastAPI and Django, wiring it into pre-commit, and running it in CI.

---

## Install ruff

```bash
bunpy add ruff --dev
```

This adds ruff to the `dev` dependency group in `pyproject.toml` and updates `uv.lock`. ruff is a development tool — it does not need to be present in the production image.

Verify the install:

```bash
bunpy run ruff --version
# ruff 0.4.x
```

---

## Running ruff

### Check for lint errors

```bash
bunpy run ruff check src/
```

ruff prints each violation with the file path, line number, rule code, and a short message:

```
src/myapp/server.py:14:5: F401 [*] `os` imported but unused
src/myapp/models.py:8:1: E302 Expected 2 blank lines, got 1
Found 2 errors.
[*] 2 fixable with the `--fix` option.
```

Fix auto-fixable violations:

```bash
bunpy run ruff check --fix src/
```

### Format code

```bash
# Check formatting without changing files
bunpy run ruff format --check src/

# Format in place
bunpy run ruff format src/
```

ruff's formatter is compatible with black. If you are migrating from black, `ruff format` produces the same output in almost all cases.

---

## pyproject.toml configuration

Add a `[tool.ruff]` section to `pyproject.toml`:

```toml
[tool.ruff]
line-length = 100
target-version = "py312"
src = ["src"]

[tool.ruff.lint]
select = [
  "E",    # pycodestyle errors
  "W",    # pycodestyle warnings
  "F",    # pyflakes
  "I",    # isort
  "B",    # flake8-bugbear
  "C4",   # flake8-comprehensions
  "UP",   # pyupgrade
  "N",    # pep8-naming
  "SIM",  # flake8-simplify
  "RUF",  # ruff-specific rules
]
ignore = [
  "E501",  # line too long — handled by formatter
  "B008",  # do not perform function calls in default arguments (conflicts with FastAPI Depends)
]

[tool.ruff.lint.isort]
known-first-party = ["myapp"]
force-sort-within-sections = true

[tool.ruff.format]
quote-style = "double"
indent-style = "space"
skip-magic-trailing-comma = false
```

### FastAPI config

FastAPI uses `Depends()` in function signatures, which triggers `B008` (function call in default argument). Ignore it:

```toml
[tool.ruff.lint]
ignore = [
  "B008",   # Depends() in function signatures
  "E501",   # line too long
]
per-file-ignores = { "tests/*" = ["S101"] }  # allow assert in tests
```

### Django config

Django uses class-based views and models that trigger naming convention rules. Adjust accordingly:

```toml
[tool.ruff.lint]
select = ["E", "W", "F", "I", "B", "C4", "UP", "RUF"]
ignore = [
  "E501",
  "N805",   # first argument of a method should be named self — conflicts with cls in classmethods
  "RUF012", # mutable class attributes (Django model fields)
]
per-file-ignores = { "*/migrations/*" = ["E501", "N806"] }
```

---

## Common fixes

**Unused imports (F401):**

```bash
bunpy run ruff check --select F401 --fix src/
```

ruff removes unused imports automatically. For `__init__.py` files that re-export names, mark them explicitly:

```python
from myapp.models import User as User  # noqa: F401
```

Or configure ruff to allow unused imports in `__init__.py`:

```toml
[tool.ruff.lint.per-file-ignores]
"__init__.py" = ["F401"]
```

**Import ordering (I001):**

```bash
bunpy run ruff check --select I --fix src/
```

ruff sorts imports according to isort conventions. It separates stdlib, third-party, and first-party imports with blank lines.

**f-string upgrades (UP032):**

```bash
bunpy run ruff check --select UP032 --fix src/
```

Rewrites `"{}".format(x)` and `"%" % x` to f-strings.

---

## pre-commit hook

pre-commit runs checks before each commit. With ruff configured, it catches issues before they reach CI.

Install pre-commit:

```bash
bunpy add pre-commit --dev
bunpy run pre-commit install
```

Create `.pre-commit-config.yaml` in the repository root:

```yaml
repos:
  - repo: https://github.com/astral-sh/ruff-pre-commit
    rev: v0.4.4
    hooks:
      - id: ruff
        args: [--fix]
      - id: ruff-format
```

The `ruff` hook runs `ruff check --fix` on staged files. The `ruff-format` hook reformats them. Both hooks run before the commit is recorded.

Test the hook manually:

```bash
bunpy run pre-commit run --all-files
```

To skip the hook for a specific commit (not recommended for normal use):

```bash
git commit --no-verify -m "WIP: skip hooks"
```

---

## VS Code extension

Install the [Ruff VS Code extension](https://marketplace.visualstudio.com/items?itemName=charliermarsh.ruff) for inline lint errors and format-on-save.

Add to `.vscode/settings.json`:

```json
{
  "[python]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "charliermarsh.ruff",
    "editor.codeActionsOnSave": {
      "source.fixAll.ruff": "explicit",
      "source.organizeImports.ruff": "explicit"
    }
  },
  "ruff.lint.args": ["--config=pyproject.toml"],
  "ruff.format.args": ["--config=pyproject.toml"]
}
```

The extension reads `pyproject.toml` for configuration. Changes to `[tool.ruff]` take effect without restarting VS Code.

---

## CI step

Add ruff to the GitHub Actions workflow (or whichever CI system you use):

```yaml
- name: Run ruff check
  run: bunpy run ruff check src/ tests/

- name: Run ruff format check
  run: bunpy run ruff format --check src/ tests/
```

The `--check` flag makes `ruff format` exit with a non-zero code if any files would be changed. This fails the CI job without modifying any files.

For the full workflow with caching, see the [CI/CD with GitHub Actions](/guides/github-actions) guide.

---

## Checking specific rules

Check a single rule to understand what it catches before adding it to `select`:

```bash
# Check only for security issues (S)
bunpy run ruff check --select S src/

# Check what pyupgrade would change
bunpy run ruff check --select UP src/

# Check with a single file for debugging
bunpy run ruff check --select B src/myapp/server.py
```

List all available rules:

```bash
bunpy run ruff rule --all
```

Show the documentation for a specific rule:

```bash
bunpy run ruff rule B008
```

---

## Migrating from flake8 + black + isort

If your project already uses these tools, ruff is a drop-in replacement:

```bash
# Remove old tools from dev dependencies
bunpy remove flake8 black isort

# Add ruff
bunpy add ruff --dev

# Run once to fix auto-fixable issues from the migration
bunpy run ruff check --fix src/
bunpy run ruff format src/
```

Delete any existing `.flake8`, `setup.cfg` `[flake8]` sections, `isort.cfg`, and `.black` config files. Move the configuration to `[tool.ruff]` in `pyproject.toml`.

The main behavioral difference: ruff's formatter uses a slightly different line-wrapping algorithm than black in a small number of cases. Run `ruff format` and review the diff before committing.
