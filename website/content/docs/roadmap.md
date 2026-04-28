---
title: Roadmap
description: Public roadmap for bunpy v0.11 and v0.12 -- documentation parity, stdlib completion, WebSocket, SQLite, and type stubs.
weight: 91
---

This page describes what is planned for v0.11.x and v0.12.x. Nothing here is a commitment; priorities can shift based on community feedback and engineering constraints. To influence what lands and when, open an issue on [GitHub](https://github.com/tamnd/bunpy/issues) and explain your use case.

The roadmap is scoped to two releases ahead. Anything further out lives in a separate notes document that changes frequently and is not published here.

## v0.11.x -- Documentation parity with bun.sh

The primary goal of v0.11 is documentation. bunpy has shipped a lot of surface area in v0.1 through v0.10, and the docs have not kept up. The v0.11 cycle is dedicated to closing that gap before adding new features.

### Target: 135+ documentation pages

bun.sh ships over 130 pages of documentation covering every CLI flag, every API, every edge case in the bundler and test runner. bunpy's current doc site has roughly 35 pages. v0.11 will bring that to at least 135 by documenting areas that are currently either missing or too thin to be useful.

The planned new pages fall into these groups:

**CLI reference (complete)**
Every command and every flag gets its own page with working examples. Currently `bunpy add`, `bunpy remove`, `bunpy build`, `bunpy test`, and `bunpy serve` each have one short page. v0.11 expands each to cover all flags, environment variables that affect the command, exit codes, and known edge cases.

**Runtime deep dives**
The import resolution algorithm -- how bunpy finds modules, in what order, and how the wheel cache interacts with local paths -- is currently described in two paragraphs. v0.11 will expand this to a full page with a resolution order diagram, examples of each case, and an explanation of what happens when a wheel and a local `.py` file share the same package name.

The globals page will be rewritten from scratch. Right now it lists `fetch`, `AbortController`, and `Bun` in one short table. The new version will cover each global in a dedicated section with type signatures, browser/Node.js compatibility notes, and common patterns.

**API reference**
`bunpy.serve`, `bunpy.fetch`, `bunpy.test`, `bunpy.file`, and `bunpy.sqlite` (the last two added in v0.12) each need a full API reference with every method, every option object, and every exception type documented.

**Guides**
Standalone how-to guides are planned for: deploying to Fly.io, deploying to Railway, using bunpy in a GitHub Actions matrix, writing a REST API, serving static files, writing a CLI tool with `argparse`, and migrating a Flask or FastAPI app to `bunpy.serve`.

**Migration guides**
v0.10 removed `bunpy.lock`. That gets a dedicated migration page. v0.11 will also document how to migrate from pip+venv, Poetry, or other Python toolchains to bunpy.

### Performance work in v0.11

One focused performance target: reduce the overhead of `bunpy test --coverage` to under 5% of total test run time. The current implementation uses a line-trace hook that serializes coverage data to disk after each file. v0.11 will replace this with an in-process coverage accumulator that flushes once at the end.

A secondary target: `bunpy build` output size. The current `.pyz` archive includes some goipy internal metadata that is not needed at runtime. Stripping it should reduce archive sizes by roughly 15% without changing the runtime behaviour.

### Stability pass

v0.11 is not a feature release, but some rough edges will be smoothed along the way:

- `bunpy serve` currently drops the connection silently if a handler raises an unhandled exception. v0.11 will log the traceback and return a 500 response instead.
- `bunpy add` with a version constraint that conflicts with an existing dependency currently prints a resolver error that references internal goipy types. The error message will be rewritten to show the conflicting constraint in plain English.
- `bunpy test --watch` occasionally misses file changes on Linux due to inotify event coalescing. A debounce fix is planned.

## v0.12.x -- Stdlib parity, WebSocket, SQLite, type stubs

v0.12 is where the runtime catches up with CPython. The goal is to have every module that matters for typical application development working correctly, with the remaining gaps limited to things that genuinely do not apply in the bunpy execution model.

### Stdlib parity: 263 of 263 modules

As of v0.10.29, goipy provides 214 of the 263 modules in the CPython 3.14 stdlib. The 49 missing modules fall into a few categories:

- **Not applicable**: `tkinter`, `turtle`, `idlelib`, and similar GUI modules have no path to a server-side interpreter. These will be documented as permanently out of scope.
- **Blocked on goipy work**: `ctypes`, `multiprocessing`, `ssl` (full implementation), `sqlite3` (stdlib wrapper; bunpy.sqlite is the replacement), and a handful of others require runtime features that goipy is still building out.
- **Low priority**: a set of rarely-used modules (`imaplib`, `nntplib`, `ossaudiodev`, etc.) that have no demand signals from the community.

The v0.12 target is 263/263 in the sense that every module either works correctly or has a documented reason it does not apply. The GUI modules and the goipy-blocked modules will be listed explicitly in the compatibility page rather than silently absent.

Modules targeted for completion in v0.12 that have real community demand: `ssl` (full TLS client and server), `hashlib` (currently missing some digest algorithms), `csv` (currently partial), `xml.etree.ElementTree` (currently partial), `configparser`, `shelve`, and `zipimport`.

### WebSocket client

`bunpy.websocket` will ship as a new built-in module. The API will mirror the browser `WebSocket` interface closely:

```python
from bunpy.websocket import WebSocket

ws = WebSocket("wss://echo.websocket.org")

@ws.on("message")
def on_message(event):
    print(event.data)

ws.send("hello")
ws.close()
```

A lower-level API for applications that need binary frames, custom ping intervals, or access to the raw close code will also be provided.

### SQLite built-in

`bunpy.sqlite` wraps the SQLite amalgamation that is already compiled into the bunpy binary. No system SQLite library is required and no extension wheel needs to be installed. The API follows the Python `sqlite3` stdlib interface closely enough that most code written against `sqlite3` will work with a one-line import change:

```python
# before
import sqlite3
db = sqlite3.connect("app.db")

# after
from bunpy.sqlite import connect
db = connect("app.db")
```

The stdlib `sqlite3` module will also be wired to bunpy.sqlite internally, so unmodified code that imports `sqlite3` will use the built-in implementation.

### Type stubs package

`bunpy-stubs` will be published to PyPI in v0.12. Installing it gives you full type information for all `bunpy.*` APIs:

```bash
bunpy add --dev bunpy-stubs
```

Type checkers that understand PEP 561 (`mypy`, `pyright`, `pytype`) will pick up the stubs automatically. The stubs will include types for `bunpy.serve`, `bunpy.fetch`, `bunpy.test`, `bunpy.file`, `bunpy.sqlite`, and `bunpy.websocket`.

## Community input

The roadmap is shaped by what people actually run into. If something important to you is missing, or if a listed item is low priority for your use case, say so on GitHub. Issues with a `roadmap` label are reviewed at the start of each release cycle. A thumbs-up on an existing issue carries the same weight as opening a new one.

Feature requests that come with a concrete use case and a proposed API get prioritised. Vague "would be nice" requests get noted but tend to stay at the bottom of the queue.
