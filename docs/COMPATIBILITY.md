# Compatibility

bunpy is a strict superset on the input side: anything `pip
install` accepts, `bunpy add` accepts. Anything `python -m pytest`
finds, `bunpy test` finds. Anything `python script.py` runs,
`bunpy script.py` runs *to the extent goipy supports the bytecode
involved*. See goipy's coverage notes for the gaps.

This page documents the boundaries.

## Python interpreter compatibility

bunpy targets **Python 3.14**. That is the version gopapy parses,
gocopy compiles to, and goipy executes. Older `.py` source still
parses (the language was largely additive across 3.x), but new
syntax from 3.15+ is not supported until the toolchain upgrades.

## C-extension wheels

goipy cannot run CPython C extensions (numpy, pillow, psycopg2,
lxml, ...) today. bunpy handles this in two ways:

1. **First-party pure-Go bindings** for the drivers people
   actually use day-to-day:
   - `bunpy.sql` substitutes for psycopg2, mysqlclient, sqlite3.
   - `bunpy.fetch` substitutes for httpx, requests, urllib3.
   - `bunpy.glob` substitutes for glob, pathlib.
   - `bunpy.WebSocket` substitutes for websockets, aiohttp's WS.
2. **Sidecar CPython, opt-in**, controlled by
   `[tool.bunpy] cpython = "auto" | "required" | "never"`. In
   `auto` (the default), bunpy runs the script on goipy and
   transparently re-exec's against CPython if a C-extension import
   is detected. `required` always uses CPython; `never` never
   does, failing fast.

The sidecar path only kicks in for the long tail; the curated
first-party set covers most production code.

## pip / pyproject.toml compatibility

- `pyproject.toml` is read per PEP 517 / 518 / 621. Project
  metadata, dependencies, optional-deps, scripts.
- `requirements.txt` is also accepted (`bunpy install -r`).
- `setup.py` projects build via PEP 517 backends.
- Editable installs use PEP 660 build hooks and `.pth` files.
- PyPI is queried via PEP 691 JSON; PEP 503 HTML is the fallback.
- PEP 508 markers are evaluated for resolver decisions.

## pytest / unittest compatibility

- `test_*.py` and `*_test.py` patterns; `tests/` directory
  walking; `conftest.py` fixture plumbing.
- `unittest.TestCase` subclasses run as-is.
- pytest fixtures with `@pytest.fixture` work.
- Tests can mix and match: `bunpy.expect(...)` matchers can be
  used inside pytest tests, and pytest fixtures can be used
  inside bunpy `describe`/`it` blocks.

## asyncio / trio / anyio

bunpy ships a Go-native asyncio policy. Existing code that uses
`asyncio.run(...)` and `async`/`await` works without changes;
when it lands, `await fetch(...)` runs on goroutines, not Python
threads. Trio and AnyIO are not re-implemented; if your code uses
them, they run on goipy as ordinary Python (no Go acceleration).

## Web-platform globals

bunpy injects `fetch`, `Request`, `Response`, `Headers`, `URL`,
`URLSearchParams`, `WebSocket`, `setTimeout`, `setInterval`,
`AbortController`, `TextEncoder`, `TextDecoder`, `structuredClone`
as Python builtins. None collide with stdlib (Python's
identifiers are case-sensitive and the stdlib does not own these
names). `bunpy --no-globals` opts out.
