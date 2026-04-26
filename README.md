<h1 align="center">bunpy</h1>

<p align="center">
  <b>Bun for Python. One binary: runtime + package manager + bundler + test runner.</b><br>
  <sub>Pure-Go Python toolchain. No CPython on the box. No cgo.</sub>
</p>

<p align="center">
  <a href="https://github.com/tamnd/bunpy/actions"><img src="https://github.com/tamnd/bunpy/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
  <img src="https://img.shields.io/badge/python-3.14-3776AB?logo=python&logoColor=white" alt="Python 3.14">
  <img src="https://img.shields.io/badge/go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go 1.26+">
</p>

---

`bunpy` is to Python what Bun is to JavaScript: one static binary
that runs Python scripts, installs PyPI packages, bundles projects
into single executables, and runs tests. No virtualenv, no
`pip install`, no separate interpreter. `bunpy app.py` works on
a fresh machine.

The runtime piece is built on the Pure-Go Python toolchain in this
ecosystem:

- [`gopapy`](https://github.com/tamnd/gopapy) parses Python 3.14
  source into an AST.
- [`gocopy`](https://github.com/tamnd/gocopy) compiles AST to a
  CPython-compatible `.pyc`.
- [`goipy`](https://github.com/tamnd/goipy) runs `.pyc` on a
  Go-native bytecode VM.

bunpy is the umbrella binary that wires those three together and
adds the package manager, bundler, test runner, dev server, and a
curated set of built-in clients (HTTP, SQL, Redis, S3, shell,
glob, cron, fetch, WebSocket).

## Status

Early bootstrap. v0.0.x rungs land one PR at a time; nothing
about the public surface is frozen until v0.5.0. Track scope and
progress in [`docs/COVERAGE.md`](docs/COVERAGE.md). The
architecture summary is in
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) and the
per-version plan is in [`docs/ROADMAP.md`](docs/ROADMAP.md).

## Quick start

```sh
go build ./cmd/bunpy
./bunpy --version
./bunpy --help
```

Once v0.0.2 lands:

```sh
echo 'print("hello from bunpy")' > /tmp/hello.py
./bunpy /tmp/hello.py
# hello from bunpy
```

## Why "Bun for Python"

Python today fragments the developer experience the way JavaScript
did before Bun: one tool to run code, several to install packages,
several more to bundle, another to test, another to format, another
to run scripts. bunpy collapses that into one binary with a
curated, fast set of defaults. Anything PyPI accepts, `bunpy add`
accepts. Anything pytest finds, `bunpy test` finds. Anything
`python script.py` runs, `bunpy script.py` runs (within goipy's
bytecode coverage; C-extension wheels are out of scope until
goipy's FFI bridge lands).

On top of that, bunpy ships the Bun-shaped surface: `bunpy.serve`,
`bunpy.sql`, `bunpy.redis`, `bunpy.s3`, `bunpy.shell`,
`bunpy.cron`, `bunpy.fetch`, `bunpy.WebSocket`, hot reload, single-
binary compile, parallel test runner. None of those need a separate
install.

## CLI map (Bun → bunpy)

| Bun                    | bunpy                  |
| ---------------------- | ---------------------- |
| `bun <file>`           | `bunpy <file>`         |
| `bun run`              | `bunpy run`            |
| `bun test`             | `bunpy test`           |
| `bun install`          | `bunpy install`        |
| `bun add <pkg>`        | `bunpy add <pkg>`      |
| `bun remove <pkg>`     | `bunpy remove <pkg>`   |
| `bun update`           | `bunpy update`         |
| `bun outdated`         | `bunpy outdated`       |
| `bun audit`            | `bunpy audit`          |
| `bun link` / `unlink`  | `bunpy link` / `unlink`|
| `bun patch <pkg>`      | `bunpy patch <pkg>`    |
| `bun publish`          | `bunpy publish`        |
| `bun pm cache rm`      | `bunpy pm cache rm`    |
| `bun why <pkg>`        | `bunpy why <pkg>`      |
| `bun init`             | `bunpy init`           |
| `bun create <tmpl>`    | `bunpy create <tmpl>`  |
| `bunx <pkg>`           | `bunpyx <pkg>`         |
| `bun build`            | `bunpy build`          |
| `bun build --compile`  | `bunpy build --compile`|
| `bun --hot`            | `bunpy run --hot`      |
| `bun --watch`          | `bunpy run --watch`    |

## Python API surface (`bunpy.*`)

A first-party `bunpy` module is registered into the runtime at
startup; no install needed. Snake-case names mirror Bun's camelCase.

```python
import bunpy

server = bunpy.serve(port=3000, fetch=lambda req: bunpy.web.Response("ok"))

text = bunpy.file("README.md").text()
db = bunpy.sql("sqlite://app.db")
rows = db("SELECT * FROM users WHERE id = ?", 1).all()

r = bunpy.redis("redis://localhost:6379")
r.set("key", "value")

s3 = bunpy.s3({"region": "us-east-1", "bucket": "logs"})
url = s3.presign("k", method="GET", expires_in=3600)

@bunpy.cron("*/5 * * * *")
def heartbeat():
    print("alive")

resp = await fetch("https://example.com")
hashed = bunpy.password.hash("hunter2")
```

The same `fetch`, `Request`, `Response`, `URL`, `Headers`,
`setTimeout`, `WebSocket`, `TextEncoder`, `AbortController` Web-
platform globals Bun injects are injected into bunpy programs.
`bunpy --no-globals` opts out.

## Tests

```sh
go test ./...        # unit tests across all packages
tests/run.sh         # end-to-end fixtures
```

`tests/run.sh` exercises `bunpy install`, `bunpy run`,
`bunpy test`, and `bunpy build --compile` on a frozen set of
example projects under `tests/fixtures/`.

## Stability

Public surface is not frozen until v0.5.0. After v0.5.0:

- **CLI surface stable.** Subcommand and flag names will not
  change without a major version bump.
- **Library entry points stable.** `bunpy.serve`, `bunpy.file`,
  `bunpy.sql`, `bunpy.test`, `bunpy.build`, the Go-side
  `runtime.Run`, `pkg.Install`, `build.Build`.
- **Module path is `github.com/tamnd/bunpy/v1`.** Future
  breaking changes move to `/v2`.

Internal helpers under `internal/` are exempt and may move freely.

## License

MIT. See [LICENSE](LICENSE).
