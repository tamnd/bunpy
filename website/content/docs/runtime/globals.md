---
title: Injected globals
description: Web-platform globals available in every bunpy script without an import statement.
weight: 3
---

```python
# No imports needed for any of these
resp = fetch("https://api.example.com/users")
u = URL("https://example.com/path?page=2")
console.log("ready")
tid = setTimeout(lambda: print("done"), 1000)
```

bunpy injects a set of web-platform globals into every script's global scope. They are available immediately, without any import. The design mirrors Bun's global injection so that code targeting Bun's runtime works in bunpy with minimal changes.

## fetch

Makes HTTP requests. Also importable as `from bunpy.fetch import fetch` if you prefer explicit imports.

```python
# GET
resp = fetch("https://api.github.com/users/octocat")
user = resp.json()
print(user["name"])  # The Octocat

# POST with JSON body
resp = fetch("https://httpbin.org/post", {
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": '{"score": 42}',
})
print(resp.status)  # 200

# With timeout (milliseconds)
resp = fetch("https://slow.example.com/data", {"timeout": 5000})
if resp.ok:
    data = resp.bytes()
```

See [bunpy.fetch](/docs/api/fetch/) for the full API reference.

## URL

Parses and manipulates URLs following the WHATWG URL Standard.

```python
u = URL("https://example.com/search?q=bunpy&page=2#results")

print(u.protocol)           # https:
print(u.hostname)           # example.com
print(u.pathname)           # /search
print(u.hash)               # #results

print(u.searchParams.get("q"))     # bunpy
print(u.searchParams.get("page"))  # 2

u.searchParams.set("page", "3")
print(str(u))  # https://example.com/search?q=bunpy&page=3#results

# Relative URL resolution
base = URL("https://example.com/docs/")
child = URL("../api/", base)
print(child.href)  # https://example.com/api/
```

## Request and Response

Construct request objects for use with `fetch`, or build response objects in handlers.

```python
# Build a Request and pass it to fetch
req = Request("https://api.example.com/data", {
    "method": "GET",
    "headers": {"Authorization": "Bearer token123"},
})
resp = fetch(req)
print(resp.status)

# Inspect a Response
print(resp.ok)            # True if 200-299
print(resp.headers)       # dict of response headers
print(resp.url)           # final URL after redirects
data = resp.json()        # parsed JSON body
raw = resp.bytes()        # raw bytes
text = resp.text()        # decoded UTF-8 string
```

## WebSocket

Client-side WebSocket. Events fire as the connection receives messages.

```python
ws = WebSocket("wss://echo.websocket.org")

ws.onopen = lambda evt: ws.send("hello from bunpy")
ws.onmessage = lambda evt: print("received:", evt.data)
ws.onclose = lambda evt: print("connection closed")
ws.onerror = lambda evt: print("error:", evt.message)

# Send later
import time
time.sleep(1)
ws.close()
```

## Timers

Browser-compatible timer functions. All delays are in milliseconds.

```python
# Fire once after a delay
tid = setTimeout(lambda: print("fired after 500ms"), 500)

# Cancel a pending timeout
clearTimeout(tid)

# Fire repeatedly
iid = setInterval(lambda: print("tick"), 1000)

# Stop the interval after 5 ticks
import threading
threading.Timer(5.0, lambda: clearInterval(iid)).start()

# Schedule a microtask (runs before the next I/O event)
queueMicrotask(lambda: print("microtask"))
```

## console

Structured logging to stderr, mirroring the browser `console` API.

```python
console.log("server started", {"port": 3000})
console.info("request received", req.method, req.path)
console.warn("rate limit approaching", {"remaining": 5})
console.error("database connection failed", err)
console.debug("query took", elapsed_ms, "ms")  # only if DEBUG=true

# console.time / console.timeEnd for quick profiling
console.time("load")
data = load_large_dataset()
console.timeEnd("load")  # Prints: load: 342ms
```

## Bun-compatible globals

These mirror Bun's API surface so that code targeting Bun runs in bunpy without changes.

| Name | Description |
|------|-------------|
| `Bun.env` | Alias for `os.environ` -- read environment variables |
| `Bun.sleep(ms)` | Async sleep for `ms` milliseconds |
| `Bun.file(path)` | Open a file handle |
| `Bun.write(path, data)` | Write `data` (bytes or str) to `path` |
| `Bun.hash(data)` | Fast non-cryptographic hash (Wyhash) |
| `Bun.serve(options)` | Start an HTTP server (same as `bunpy.serve`) |
| `Bun.version` | bunpy version string |
| `Bun.main` | Path to the entry-point script |

```python
# Read an env variable
db_url = Bun.env.get("DATABASE_URL", "sqlite:///dev.db")

# Write a file
Bun.write("/tmp/output.txt", "hello\n")

# Read it back
f = Bun.file("/tmp/output.txt")
print(f.text())  # hello

# Hash a string
h = Bun.hash("some data")
print(h)  # integer
```

## process

Node.js-compatible `process` object. Useful for code that targets both Node.js and bunpy.

```python
import sys

print(process.argv)       # same as sys.argv
print(process.env)        # same as os.environ
print(process.pid)        # current process ID
print(process.platform)   # "linux", "darwin", "windows"
print(process.version)    # "v3.14.0"

process.exit(0)           # same as sys.exit(0)
process.exit(1)           # exit with error code

# Event callbacks
process.on("exit", lambda code: print(f"exiting with code {code}"))
process.on("uncaughtException", lambda err: print("unhandled:", err))
```

## structuredClone

Deep-clone any serialisable object:

```python
original = {"a": [1, 2, 3], "b": {"nested": True}}
clone = structuredClone(original)
clone["a"].append(4)
print(original["a"])  # [1, 2, 3] -- unchanged
print(clone["a"])     # [1, 2, 3, 4]
```

Works with dicts, lists, strings, numbers, booleans, and `None`. Raises `TypeError` for objects that cannot be serialised (functions, class instances without `__dict__`, etc.).

## crypto

Global access to the Web Crypto API subset:

```python
# Generate random bytes
random_bytes = crypto.getRandomValues(bytearray(16))
print(random_bytes.hex())

# Random UUID
uid = crypto.randomUUID()
print(uid)  # "f47ac10b-58cc-4372-a567-0e02b2c3d479"
```

For full cryptographic operations (hashing, HMAC, key generation), use the `bunpy.crypto` module instead.
