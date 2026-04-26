# CLI reference

bunpy ships as one binary. Subcommands land per-version per the
roadmap; v0.0.1 only has `--version` and `--help`. This page is
the long-form reference. Running `bunpy help <cmd>` gives the
short form.

## Synopsis

```
bunpy <command> [args]
bunpy --version
bunpy --help
```

A bare positional `.py` argument is shorthand for `bunpy run
<file>`. So `bunpy app.py` and `bunpy run app.py` mean the same
thing.

## Commands

### Runtime

- `bunpy run <file.py>` runs a Python script. Flags: `--hot`
  (state-preserving reload), `--watch` (full restart on change),
  `-` (read from stdin), `--no-globals` (do not inject Web
  platform globals).
- `bunpy repl` opens an interactive REPL. History at
  `~/.bunpy_history`.

### Package manager

- `bunpy install` installs dependencies from `pyproject.toml`
  and `bunpy.lock`. `--frozen` refuses to mutate the lockfile.
- `bunpy add <pkg>` adds a dependency. `-D` for dev, `-O` for
  optional, `-P` for peer.
- `bunpy remove <pkg>` removes a dependency.
- `bunpy update [pkg]` updates one or all packages.
- `bunpy outdated [pkg]` lists packages with newer versions.
- `bunpy audit [--fix]` checks for security advisories.
- `bunpy link [pkg]` and `bunpy unlink [pkg]` do editable
  installs.
- `bunpy patch <pkg>` and `bunpy patch --commit <hash>` persist
  local diffs against installed packages.
- `bunpy publish` builds an sdist plus a wheel and uploads them
  to PyPI.
- `bunpy why <pkg>` prints a reverse-deps tree explaining why a
  package is in the lockfile.
- `bunpy pm cache rm` clears on-disk caches.
- `bunpy pm ls` lists installed packages.
- `bunpy pm hash` prints the lockfile content hash.

### Project scaffolding

- `bunpy init` scaffolds `pyproject.toml`, the src layout, and a
  README.
- `bunpy create <template>` scaffolds from a template (fastapi,
  flask, click, lib, ml).
- `bunpyx <pkg>[@version] [args]` does a one-shot run from PyPI.

### Bundler

- `bunpy build [<entry.py>]` bundles to a `.pyz` (default).
- `bunpy build --compile` bundles into a single static Go binary.
- `bunpy build --target <triple>` produces a cross-target bundle.
- `bunpy build --plugins <list>` runs with bundler plugins.

### Test runner

- `bunpy test [path]` discovers and runs tests. Flags include
  `--parallel[=N]`, `--isolate`, `--shard=I/N`, `--changed`,
  `--coverage[=html|json|lcov]`, `--watch`,
  `--update-snapshots`, `--bail[=N]`, `--timeout=<ms>`.

### Workspace selectors

- `bunpy --filter <selector> <command>` runs a command in a
  matching workspace subset.

### Tooling passthrough

- `bunpy fmt [path]` formats Python source via gopapy.
- `bunpy check [path]` lints Python source via gopapy.

### Meta

- `bunpy version` and `--version` print the version.
- `bunpy help [cmd]` and `--help` print help.
