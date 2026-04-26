# Deviations from Bun

bunpy aims for 100% Bun feature parity for Python. Some Bun
features have no Python-shaped equivalent or have a different
shape because Python's ecosystem already has a strong convention.
Each deviation is documented here so users coming from Bun can
find the bunpy answer fast.

## Skipped (no equivalent)

### JSX / TSX loaders

JSX is JavaScript-specific. Python has no equivalent syntax. The
closest match is **PEP 750 t-strings**: tagged template literals
introduced in Python 3.14, which let a library implement
React-like syntax via `t"<div>...</div>"`. bunpy's bundler runs
t-string handlers at build time, but does not invent a JSX
dialect for Python.

### `bun:sqlite` C binding

Bun ships a sqlite driver as a C binding. bunpy ships a
**pure-Go** sqlite driver under `bunpy.sql`. The same surface
(`db("SELECT ...").all()`), different implementation. Pure-Go
keeps the single-binary promise.

### `Bun.bunPath` / `import.meta.dir` JS path conventions

Python has its own conventions: `__file__`, `pathlib.Path`,
`importlib.resources`. bunpy uses those rather than inventing a
`bunpy.bun_path` to match.

## Renamed (camelCase → snake_case)

Bun is JavaScript and uses camelCase. Python is snake_case. Every
exported name is mapped:

| Bun                  | bunpy                  |
|----------------------|------------------------|
| `Bun.deepEquals`     | `bunpy.deep_equals`    |
| `Bun.escapeHTML`     | `bunpy.escape_html`    |
| `Bun.randomUUIDv7`   | `bunpy.random_uuid_v7` |
| `Bun.fileURLToPath`  | `bunpy.file_url_to_path` |
| `Bun.pathToFileURL`  | `bunpy.path_to_file_url` |
| `Bun.openInEditor`   | `bunpy.open_in_editor` |
| `Bun.generateCryptoKey` | `bunpy.generate_crypto_key` |

Class-shaped names stay PascalCase (`bunpy.Worker`,
`bunpy.WebSocket`, `bunpy.WebView`, `bunpy.HTMLRewriter`,
`bunpy.URLPattern`, `bunpy.Terminal`).

## Reshaped (different surface, same intent)

### `Bun.$` tagged template

JavaScript supports tagged template literals; Python does not (PEP
750 t-strings come close but require a tag function). bunpy
exposes `bunpy.dollar(...)` and an alias `from bunpy import
dollar as $` so user code can write `$("git status").text()`.

### `Bun.serve` route handlers

Bun's route value can be a function or an object keyed by HTTP
verb. bunpy accepts the same shape, with Python callables and a
dict keyed by verb. Optional `Bun.match` syntax via `:param` is
supported as-is.

### `bun:test` matchers

Jest-style matchers stay chainable but follow Python naming:
`.to_be`, `.to_equal`, `.to_throw`, `.to_have_been_called_with`.
The negation prefix is `.not_to_*` rather than `.not.to*`, since
`.not` is not a valid Python attribute. The pytest assertion
shape (`assert x == y`) is also fully supported.

### Lockfile encoding

Bun's `bun.lock` is a binary format. bunpy's `bunpy.lock` is
TOML, PEP 751-shaped, sorted, human-readable. Trade-off:
slightly slower to parse, but diff-friendly and reviewable.

## Carried forward

Almost everything else is a 1:1 carry. The CLI map in
`docs/CLI.md`, the API surface in `docs/API.md`, and the coverage
table in `docs/COVERAGE.md` are the canonical references.
