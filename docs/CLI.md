# CLI reference

bunpy ships as one binary. Subcommands land per-version per the
roadmap; v0.0.1 only has `--version` and `--help`. This page is
the long-form reference; running `bunpy help <cmd>` gives the
short form.

## Synopsis

```
bunpy <command> [args]
bunpy --version
bunpy --help
```

A bare positional `.py` argument is shorthand for `bunpy run
<file>` — `bunpy app.py` is the same as `bunpy run app.py`.

## Commands

### Runtime

- `bunpy run <file.py>` — Run a Python script. Flags: `--hot` (state-
  preserving reload), `--watch` (full restart on change), `-` (read
  from stdin), `--no-globals` (do not inject Web-platform globals).
- `bunpy repl` — Interactive REPL. History at `~/.bunpy_history`.

### Package manager

- `bunpy install` — Install dependencies from `pyproject.toml` +
  `bunpy.lock`. `--frozen` refuses to mutate the lockfile.
- `bunpy add <pkg>` — Add a dependency. `-D` dev, `-O` optional,
  `-P` peer.
- `bunpy remove <pkg>` — Remove a dependency.
- `bunpy update [pkg]` — Update one or all packages.
- `bunpy outdated [pkg]` — List packages with newer versions.
- `bunpy audit [--fix]` — Check for security advisories.
- `bunpy link [pkg]` / `bunpy unlink [pkg]` — Editable install.
- `bunpy patch <pkg>` / `bunpy patch --commit <hash>` — Persist
  local diffs against installed packages.
- `bunpy publish` — Build sdist + wheel and upload to PyPI.
- `bunpy why <pkg>` — Reverse-deps tree explaining why a package
  is in the lockfile.
- `bunpy pm cache rm` — Clear on-disk caches.
- `bunpy pm ls` — List installed packages.
- `bunpy pm hash` — Print the lockfile content hash.

### Project scaffolding

- `bunpy init` — Scaffold pyproject.toml + src layout + README.
- `bunpy create <template>` — Scaffold from a template (fastapi,
  flask, click, lib, ml).
- `bunpyx <pkg>[@version] [args]` — One-shot run from PyPI.

### Bundler

- `bunpy build [<entry.py>]` — Bundle to a `.pyz` (default).
- `bunpy build --compile` — Bundle into a single static Go
  binary.
- `bunpy build --target <triple>` — Cross-target bundles.
- `bunpy build --plugins <list>` — Run with bundler plugins.

### Test runner

- `bunpy test [path]` — Discover and run tests. Flags:
  `--parallel[=N]`, `--isolate`, `--shard=I/N`, `--changed`,
  `--coverage[=html|json|lcov]`, `--watch`, `--update-snapshots`,
  `--bail[=N]`, `--timeout=<ms>`.

### Workspace selectors

- `bunpy --filter <selector> <command>` — Run a command in a
  matching workspace subset.

### Tooling passthrough

- `bunpy fmt [path]` — Format Python source via gopapy.
- `bunpy check [path]` — Lint Python source via gopapy.

### Meta

- `bunpy version` / `--version` — Print version.
- `bunpy help [cmd]` / `--help` — Print help.
