---
title: CLI reference
description: Every bunpy command, flag, and option.
weight: 4
---

bunpy ships one binary that replaces `python3`, `pip`, `pytest`, `black`,
`ruff`, and a bundler. All functionality is exposed through subcommands.

## Global flags

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Print help for the command |
| `--version`, `-v` | Print bunpy version and exit |

## Commands

{{< cards >}}
  {{< card link="run" title="run" subtitle="Execute a Python script" >}}
  {{< card link="install" title="install" subtitle="Install project dependencies" >}}
  {{< card link="add" title="add" subtitle="Add a package to pyproject.toml" >}}
  {{< card link="remove" title="remove" subtitle="Remove a package" >}}
  {{< card link="update" title="update" subtitle="Upgrade locked packages" >}}
  {{< card link="build" title="build" subtitle="Bundle a script to .pyz or native binary" >}}
  {{< card link="test" title="test" subtitle="Run tests" >}}
  {{< card link="fmt" title="fmt" subtitle="Format Python source" >}}
  {{< card link="check" title="check" subtitle="Lint / static type check" >}}
  {{< card link="repl" title="repl" subtitle="Interactive Python REPL" >}}
  {{< card link="create" title="create" subtitle="Scaffold a new project from a template" >}}
  {{< card link="publish" title="publish" subtitle="Publish a package to PyPI" >}}
  {{< card link="version" title="version" subtitle="Print version information" >}}
{{< /cards >}}
