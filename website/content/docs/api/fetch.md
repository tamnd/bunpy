---
title: bunpy.fetch
description: HTTP client global and module -- make requests to any HTTP or HTTPS endpoint.
weight: 1
---

```python
resp = fetch("https://api.github.com/users/octocat")
user = resp.json()
print(user["name"])  # The Octocat
```

`fetch` is injected into every bunpy script as a global. No import is needed. For explicit imports:

```python
from bunpy.fetch import fetch
```

Both are the same function. The global is there for convenience; the import is there for clarity and type-checker support.

## Signature

```python
fetch(url: str | Request, options: dict = {}) -> Response
```

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `method` | str | `"GET"` | HTTP method |
| `headers` | dict | `{}` | Request headers |
| `body` | str or bytes | `None` | Request body |
| `redirect` | str | `"follow"` | `"follow"`, `"manual"`, or `"error"` |
| `timeout` | int | `30000` | Timeout in milliseconds. `0` disables the timeout. |
| `keepalive` | bool | `False` | Keep the connection alive for reuse |

## Response

| Attribute / Method | Type | Description |
|--------------------|------|-------------|
| `.status` | int | HTTP status code |
| `.status_code` | int | Alias for `.status` |
| `.ok` | bool | `True` if `200 <= status < 300` |
| `.headers` | dict | Response headers (lowercase keys) |
| `.url` | str | Final URL after redirects |
| `.redirected` | bool | `True` if at least one redirect occurred |
| `.text()` | str | Decode body as UTF-8 |
| `.json()` | any | Parse body as JSON |
| `.bytes()` | bytes | Raw body bytes |

## Examples

### GET request

```python
resp = fetch("https://api.github.com/repos/tamnd/bunpy")
repo = resp.json()
print(repo["stargazers_count"])
print(repo["description"])
```

### POST with JSON

```python
resp = fetch("https://httpbin.org/post", {
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": '{"username": "alice", "score": 99}',
})

if resp.ok:
    data = resp.json()
    print(data["json"])  # {"username": "alice", "score": 99}
```

### POST with a Python dict (auto-serialised)

```python
import json

payload = {"items": [1, 2, 3], "total": 6}
resp = fetch("https://httpbin.org/post", {
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": json.dumps(payload),
})
print(resp.status)  # 200
```

### POST form data

```python
import urllib.parse

form = urllib.parse.urlencode({"username": "alice", "password": "secret"})
resp = fetch("https://example.com/login", {
    "method": "POST",
    "headers": {"Content-Type": "application/x-www-form-urlencoded"},
    "body": form,
})
print(resp.status)
```

### Custom headers and authentication

```python
import os

token = os.environ["GITHUB_TOKEN"]
resp = fetch("https://api.github.com/user", {
    "headers": {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github.v3+json",
    }
})
print(resp.json()["login"])
```

### Timeout

```python
try:
    resp = fetch("https://slow.example.com/data", {"timeout": 3000})
except TimeoutError:
    print("request timed out after 3 seconds")
```

### Following vs. blocking redirects

```python
# Default: follow redirects automatically
resp = fetch("https://httpbin.org/redirect/3")
print(resp.redirected)  # True
print(resp.url)         # final URL

# Manual: stop at the first redirect
resp = fetch("https://httpbin.org/redirect/3", {"redirect": "manual"})
print(resp.status)  # 302

# Error on redirect
try:
    resp = fetch("https://httpbin.org/redirect/3", {"redirect": "error"})
except Exception as e:
    print(e)  # redirect was disallowed
```

### Read response as bytes

```python
resp = fetch("https://via.placeholder.com/150")
image_bytes = resp.bytes()
with open("image.png", "wb") as f:
    f.write(image_bytes)
print(f"saved {len(image_bytes)} bytes")
```

### Check response headers

```python
resp = fetch("https://httpbin.org/get")
content_type = resp.headers.get("content-type", "")
print(content_type)  # application/json
```

### Reuse a Request object

```python
from bunpy.fetch import fetch

req = Request("https://api.example.com/data", {
    "method": "GET",
    "headers": {"Authorization": "Bearer token123"},
})

# Fetch it once
resp1 = fetch(req)

# Fetch again (same request object)
resp2 = fetch(req)
```

## Error handling

`fetch` raises exceptions for network-level errors. HTTP errors (4xx, 5xx) do not raise -- check `.ok` or `.status` yourself:

```python
try:
    resp = fetch("https://api.example.com/data")
except ConnectionError as e:
    print("network error:", e)
except TimeoutError:
    print("timed out")
else:
    if not resp.ok:
        print(f"HTTP {resp.status}: {resp.text()}")
    else:
        data = resp.json()
```

## Async usage

`fetch` is synchronous by default. In `async def` contexts, use `await fetch(...)`:

```python
import asyncio

async def load_users():
    resp = await fetch("https://api.example.com/users")
    return resp.json()

users = asyncio.run(load_users())
print(users)
```

The async form yields control to the event loop while waiting for the response, allowing other coroutines to run concurrently.
