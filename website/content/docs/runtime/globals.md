---
title: Injected globals
description: Web-platform globals available in every bunpy script without import.
weight: 3
---

bunpy injects the following names into the global scope of every script.
No import statement is needed.

## fetch

```python
resp = fetch("https://httpbin.org/get")
resp = fetch("https://api.example.com/data", {
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": '{"key": "value"}',
})
```

Returns a `Response`-like object with `.status_code`, `.text()`, `.json()`,
and `.bytes()` methods.

## URL

```python
u = URL("https://example.com/path?q=1")
print(u.hostname)   # example.com
print(u.pathname)   # /path
print(u.searchParams.get("q"))  # 1
```

## Request / Response

```python
req = Request("https://example.com", method="GET")
resp = fetch(req)
```

## WebSocket

```python
ws = WebSocket("wss://echo.websocket.org")
ws.onmessage = lambda evt: print(evt.data)
ws.send("hello")
```

## Timers

```python
tid = setTimeout(lambda: print("fired!"), 1000)
clearTimeout(tid)

iid = setInterval(lambda: print("tick"), 500)
clearInterval(iid)
```

## console

```python
console.log("hello", "world")
console.error("something went wrong")
console.warn("watch out")
```

## Bun-compatible globals

| Name | Description |
|------|-------------|
| `Bun.env` | Alias for `os.environ` |
| `Bun.sleep(ms)` | Async sleep for `ms` milliseconds |
| `Bun.file(path)` | Open a file handle |
| `Bun.write(path, data)` | Write bytes or string to a file |
| `Bun.hash(data)` | Fast non-cryptographic hash (Wyhash) |
| `Bun.serve(options)` | Start an HTTP server |

## process

```python
import sys
print(process.argv)     # same as sys.argv
print(process.env)      # same as os.environ
process.exit(0)         # same as sys.exit(0)
```
