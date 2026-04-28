---
title: "bunpy now has a website"
date: 2026-04-28
description: We shipped tamnd.github.io/bunpy - full documentation, CLI reference, API docs, and guides.
---

bunpy v0.9.x ships a full documentation website at
[tamnd.github.io/bunpy](https://tamnd.github.io/bunpy).

The site mirrors the bun.sh content structure: installation guide, quickstart,
CLI reference for every subcommand, runtime internals, bundler docs, test
runner docs, package manager docs, full API reference for all `bunpy.*` and
`bunpy.node.*` modules, and a guides section.

## What's there

- **Getting started** - install in one curl command, write your first script
- **CLI reference** - every flag for every subcommand
- **Runtime docs** - Python 3.14 compat, import resolution, injected globals,
  Node.js API shims, goipy VM internals
- **Bundler docs** - `.pyz` format, cross-compile targets, watch mode
- **Test runner** - `bunpy test`, mocking, snapshots
- **Package manager** - `bunpy install/add/remove/update`, workspaces, lockfile
- **API reference** - 14 `bunpy.*` modules + 6 `bunpy.node.*` modules
- **Guides** - HTTP server, CLI app, Docker deployment

## Built with Hugo + hextra

The site is built with [Hugo](https://gohugo.io/) and the
[hextra](https://github.com/imfing/hextra) theme. It deploys automatically via
GitHub Actions on every push to `main`.

Source: [github.com/tamnd/bunpy](https://github.com/tamnd/bunpy) in the
`website/` directory.
