---
title: Compatibility
description: Python stdlib coverage, Python version requirements, platform support, and Bun API availability in bunpy as of v0.10.x.
weight: 92
---

This page documents what bunpy supports as of v0.10.29 and what it does not. Coverage numbers are updated each release; check the [changelog](/docs/changelog/) for what changed.

## Python version compatibility

bunpy embeds goipy, a Go-native CPython 3.14 interpreter. There is no system Python involved.

| Property | Status |
|----------|--------|
| Syntax | Python 3.12+ required. f-strings with nested expressions (PEP 701), `match`/`case` (PEP 634), exception groups (PEP 654), `type` aliases (PEP 695) all work. |
| Semantics | CPython 3.14 target. goipy tracks the 3.14 reference implementation for semantics and standard library behaviour. |
| Python 2 | Not supported and never will be. |
| Python 3.11 and below | Syntax that was removed or changed in 3.12+ will error. |
| Type annotations | Supported. PEP 563 (`from __future__ import annotations`) and PEP 649 (lazy annotations, 3.14 default) both work. |
| C extension modules | Not supported. Packages that require compiled `.so` or `.pyd` extension wheels cannot be imported. Pure-Python wheels work. |

If your code runs on CPython 3.12 or 3.13 without C extensions, it will almost certainly run on bunpy with no changes.

## Python stdlib coverage

214 of 263 CPython 3.14 stdlib modules are supported as of v0.10.29. The tables below show coverage by category. Modules with partial support have the most commonly-used parts working; the gaps are documented in each module's page.

### Built-in types and core language

| Module | Status | Notes |
|--------|--------|-------|
| `builtins` | Supported | All built-in functions and types. |
| `abc` | Supported | |
| `collections` | Supported | All: `deque`, `Counter`, `OrderedDict`, `defaultdict`, `namedtuple`, `ChainMap`. |
| `collections.abc` | Supported | |
| `dataclasses` | Supported | |
| `enum` | Supported | |
| `functools` | Supported | |
| `itertools` | Supported | |
| `operator` | Supported | |
| `types` | Supported | |
| `typing` | Supported | |
| `typing_extensions` | Partial | Not stdlib; available as a PyPI package. |
| `weakref` | Supported | |

### Text and string processing

| Module | Status | Notes |
|--------|--------|-------|
| `re` | Supported | |
| `string` | Supported | |
| `textwrap` | Supported | |
| `unicodedata` | Supported | |
| `difflib` | Supported | |
| `fnmatch` | Supported | |
| `glob` | Supported | |

### File and I/O

| Module | Status | Notes |
|--------|--------|-------|
| `io` | Supported | |
| `os` | Supported | `os.fork` not available (goroutine model). |
| `os.path` | Supported | |
| `pathlib` | Supported | |
| `shutil` | Supported | |
| `tempfile` | Supported | |
| `stat` | Supported | |
| `fileinput` | Supported | |
| `csv` | Partial | Reading works; writer missing quoting edge cases. Full support in v0.12. |
| `configparser` | Partial | Core read/write works; interpolation incomplete. Full support in v0.12. |
| `zipfile` | Supported | |
| `tarfile` | Supported | |
| `gzip` | Supported | |
| `bz2` | Supported | |
| `lzma` | Supported | |
| `pickle` | Supported | |
| `shelve` | Partial | Full support targeted for v0.12. |
| `struct` | Supported | |
| `mmap` | Not supported | Requires OS memory-mapping primitives not yet wired in goipy. |

### Networking

| Module | Status | Notes |
|--------|--------|-------|
| `socket` | Partial | TCP and UDP sockets work. Unix domain sockets work on macOS and Linux. Raw sockets not supported. |
| `ssl` | Partial | TLS client connections work. Server-side TLS and client certificate authentication incomplete. Full support in v0.12. |
| `http.client` | Supported | |
| `http.server` | Supported | Use `bunpy.serve` for production; `http.server` is available for compatibility. |
| `urllib.request` | Supported | |
| `urllib.parse` | Supported | |
| `email` | Supported | |
| `smtplib` | Supported | |
| `ftplib` | Supported | |
| `imaplib` | Not supported | Low demand; file an issue if you need it. |
| `poplib` | Not supported | Low demand. |
| `nntplib` | Not supported | Deprecated in CPython 3.13; will not be added. |
| `xmlrpc.client` | Supported | |
| `xmlrpc.server` | Supported | |
| `websockets` | Not built-in | Use the `websockets` PyPI package (pure Python). `bunpy.websocket` built-in planned for v0.12. |

### Data formats and serialization

| Module | Status | Notes |
|--------|--------|-------|
| `json` | Supported | |
| `xml.etree.ElementTree` | Partial | Basic parsing and serialization work; XPath subset incomplete. Full support in v0.12. |
| `xml.dom` | Partial | |
| `xml.sax` | Partial | |
| `html` | Supported | |
| `html.parser` | Supported | |
| `base64` | Supported | |
| `hashlib` | Partial | MD5, SHA-1, SHA-256, SHA-512 work. SHA-3 and BLAKE2 variants missing. Full suite in v0.12. |
| `hmac` | Supported | |
| `binascii` | Supported | |

### Date, time, and math

| Module | Status | Notes |
|--------|--------|-------|
| `datetime` | Supported | |
| `time` | Supported | |
| `calendar` | Supported | |
| `math` | Supported | |
| `cmath` | Supported | |
| `decimal` | Supported | |
| `fractions` | Supported | |
| `random` | Supported | |
| `statistics` | Supported | |

### Concurrency

| Module | Status | Notes |
|--------|--------|-------|
| `threading` | Supported | Threads map to goroutines. The GIL does not exist in goipy; actual parallelism is available. |
| `asyncio` | Supported | Event loop, coroutines, tasks, `asyncio.gather`, `asyncio.sleep`, streams. |
| `concurrent.futures` | Supported | `ThreadPoolExecutor` and `ProcessPoolExecutor` both work. |
| `multiprocessing` | Partial | `Process`, `Queue`, `Pipe` work. Shared memory (`multiprocessing.shared_memory`) not yet supported. |
| `queue` | Supported | |
| `subprocess` | Supported | |

### Databases

| Module | Status | Notes |
|--------|--------|-------|
| `sqlite3` | Partial | Wired to `bunpy.sqlite` internally. Full parity with CPython's `sqlite3` targeted for v0.12. |
| `dbm` | Not supported | |

### Development and introspection

| Module | Status | Notes |
|--------|--------|-------|
| `sys` | Supported | `sys.platform`, `sys.argv`, `sys.path`, `sys.version`, `sys.exit`, and most others. `sys.settrace` and `sys.setprofile` work (used by the coverage tool). |
| `inspect` | Supported | |
| `traceback` | Supported | |
| `warnings` | Supported | |
| `logging` | Supported | |
| `unittest` | Supported | All of `unittest` works. `bunpy test` is preferred but not required. |
| `pdb` | Partial | Basic breakpoint and stepping. TUI line-editor not available in all terminals. |
| `profile` / `cProfile` | Supported | |
| `dis` | Supported | Disassembles to goipy's internal bytecode format, not CPython bytecode. |
| `ast` | Supported | |
| `tokenize` | Supported | |
| `importlib` | Supported | `importlib.import_module`, `importlib.util`, `importlib.resources` all work. |

### GUI and multimedia (not supported)

`tkinter`, `turtle`, `idlelib`, `curses`, `readline`, `ossaudiodev`, `winsound` -- these modules are permanently out of scope. bunpy is a server-side runtime. If you need a GUI, use a separate Python install.

## Platform support

| Platform | Architecture | Status | Notes |
|----------|-------------|--------|-------|
| macOS | arm64 (Apple Silicon) | Supported | Primary development platform. |
| macOS | x86_64 | Supported | Tested on every release. |
| Linux | x86_64 | Supported | Tested in CI. |
| Linux | arm64 | Supported | Tested in CI; used for AWS Graviton and Raspberry Pi deployments. |
| Windows | x86_64 | Supported | PowerShell install, GitHub Actions tested. |
| Windows | arm64 | Not supported | No demand signal yet. File an issue if you need it. |
| FreeBSD | any | Not supported | |

All platforms receive binary releases on every tagged commit starting from v0.10.29.

## Bun API coverage

bunpy exposes a subset of the Bun global APIs. The goal is not full compatibility with Bun's JavaScript API surface -- it is to give Python code the same ergonomic primitives that make Bun pleasant to use.

### Globals

| Global | Status | Notes |
|--------|--------|-------|
| `fetch` | Supported | Full WHATWG Fetch API. Returns a `Response` with `.text()`, `.json()`, `.bytes()`, `.status`, `.headers`, `.ok`. |
| `AbortController` | Supported | |
| `AbortSignal` | Supported | `AbortSignal.timeout(ms)` supported. |
| `URL` | Supported | WHATWG URL standard. |
| `URLSearchParams` | Supported | |
| `TextEncoder` / `TextDecoder` | Supported | |
| `setTimeout` / `setInterval` | Supported | Available in async contexts and in `bunpy.serve` handlers. |
| `clearTimeout` / `clearInterval` | Supported | |
| `console` | Supported | `console.log`, `console.error`, `console.warn`, `console.time`, `console.timeEnd`. |
| `crypto` | Partial | `crypto.randomUUID()` and `crypto.getRandomValues()` work. SubtleCrypto not yet available. |
| `performance` | Partial | `performance.now()` works. `performance.mark` and `performance.measure` not yet available. |

### Bun namespace

| API | Status | Notes |
|-----|--------|-------|
| `Bun.serve(options)` | Supported | Available via `from bunpy.serve import serve`. The `Bun.serve` spelling is also accepted. |
| `Bun.file(path)` | Supported | Returns a lazy file handle. `.text()`, `.json()`, `.bytes()`, `.stream()`, `.size`, `.type`. |
| `Bun.write(path, data)` | Supported | Writes a string, bytes, or `Bun.file` to disk. |
| `Bun.spawn(cmd)` | Supported | Spawns a subprocess. `.stdout`, `.stderr`, `.stdin` are streams. `.exited` is an awaitable. |
| `Bun.which(name)` | Supported | Resolves an executable name to its full path. |
| `Bun.env` | Supported | Dict-like access to environment variables. |
| `Bun.version` | Supported | Returns the bunpy version string. |
| `Bun.hash(data)` | Supported | Wyhash of a string or bytes. |
| `Bun.sleep(ms)` | Supported | Awaitable sleep. `await Bun.sleep(100)`. |
| `Bun.gc()` | Supported | Triggers a GC cycle on the goipy heap. |
| `Bun.inspect(value)` | Supported | Pretty-prints any value to a string, similar to `repr` but richer. |
| `Bun.build(options)` | Not supported | JavaScript bundler API; not applicable to Python. |
| `Bun.plugin(plugin)` | Not supported | JavaScript plugin system; not applicable. |
| `Bun.websocket` | Not supported | Planned for v0.12 via `bunpy.websocket`. |
| `Bun.sql` | Not supported | Planned for v0.12 via `bunpy.sqlite`. |

### What is intentionally absent

Some Bun APIs exist specifically because JavaScript/TypeScript has module systems and compilation models that do not apply to Python. `Bun.Transpiler`, `Bun.resolveSync`, `import.meta`, and the `--preload` mechanism have no Python equivalent. They are absent by design, not by oversight.
