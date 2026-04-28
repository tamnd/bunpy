---
title: bunpy.serve
description: Built-in HTTP server -- route requests, return responses, and serve static files without any framework.
weight: 2
---

```python
from bunpy.serve import serve

def handler(req):
    return f"Hello, {req.query.get('name', 'world')}!"

serve(handler, port=3000)
```

```bash
bunpy server.py
# Listening on http://localhost:3000
```

`bunpy.serve` is an HTTP server built directly into the runtime. It handles routing, parsing, and response serialisation in Go -- no WSGI, no ASGI, no framework required. For simple services and APIs it is the fastest path from code to running server.

## Signature

```python
from bunpy.serve import serve

serve(handler, port=3000, hostname="127.0.0.1", development=False)
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `port` | int | `3000` | TCP port to bind |
| `hostname` | str | `"127.0.0.1"` | Bind address. Use `"0.0.0.0"` to accept external connections. |
| `development` | bool | `False` | Enable verbose request/response logging |
| `tls` | dict | `None` | TLS config: `{"cert": "cert.pem", "key": "key.pem"}` |
| `max_request_size` | int | `10_485_760` | Maximum body size in bytes (default 10 MB) |

## Request object

Every call to `handler` receives a `Request` object:

| Attribute | Type | Description |
|-----------|------|-------------|
| `req.method` | str | HTTP method (`GET`, `POST`, `PUT`, `DELETE`, ...) |
| `req.url` | str | Full request URL including scheme and host |
| `req.path` | str | URL path component (no query string) |
| `req.query` | dict | Parsed query parameters (first value per key) |
| `req.headers` | dict | Request headers (lowercase keys) |
| `req.body` | bytes | Raw request body |
| `req.json()` | any | Parse body as JSON. Raises `ValueError` if body is not valid JSON. |
| `req.text()` | str | Decode body as UTF-8 |
| `req.form()` | dict | Parse `application/x-www-form-urlencoded` body |

## Returning a response

The handler can return a string, bytes, dict, or a `Response` object.

### String (plain text, status 200)

```python
def handler(req):
    return "Hello!"
```

### Dict

```python
def handler(req):
    return {
        "status": 200,
        "headers": {"Content-Type": "application/json"},
        "body": '{"ok": true}',
    }
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `status` | int | `200` | HTTP status code |
| `headers` | dict | `{}` | Response headers |
| `body` | str or bytes | `""` | Response body |

### Response object

```python
from bunpy.serve import serve, Response

def handler(req):
    return Response(
        body='{"ok": true}',
        status=200,
        headers={"Content-Type": "application/json"},
    )

serve(handler, port=3000)
```

## Examples

### JSON API

```python
import json
from bunpy.serve import serve

users = [
    {"id": 1, "name": "Alice"},
    {"id": 2, "name": "Bob"},
]

def handler(req):
    if req.path == "/users" and req.method == "GET":
        return {
            "status": 200,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps(users),
        }

    if req.path == "/users" and req.method == "POST":
        data = req.json()
        new_user = {"id": len(users) + 1, "name": data["name"]}
        users.append(new_user)
        return {
            "status": 201,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps(new_user),
        }

    return {"status": 404, "body": "not found"}

serve(handler, port=3000)
```

### Path routing

```python
import json
from bunpy.serve import serve

def handler(req):
    parts = req.path.strip("/").split("/")

    if parts[0] == "health":
        return {"status": 200, "body": "ok"}

    if parts[0] == "users" and len(parts) == 2:
        user_id = int(parts[1])
        return {
            "status": 200,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps({"id": user_id, "name": f"User {user_id}"}),
        }

    return {"status": 404, "body": "not found"}

serve(handler, port=3000)
```

### Middleware pattern

```python
import time
import json
from bunpy.serve import serve

def log_middleware(handler):
    def wrapped(req):
        start = time.monotonic()
        resp = handler(req)
        elapsed = (time.monotonic() - start) * 1000
        print(f"{req.method} {req.path} -> {resp.get('status', 200)} ({elapsed:.1f}ms)")
        return resp
    return wrapped

def auth_middleware(handler):
    def wrapped(req):
        token = req.headers.get("authorization", "")
        if not token.startswith("Bearer "):
            return {"status": 401, "body": "unauthorized"}
        return handler(req)
    return wrapped

def app(req):
    return {
        "status": 200,
        "headers": {"Content-Type": "application/json"},
        "body": json.dumps({"message": "protected resource"}),
    }

serve(log_middleware(auth_middleware(app)), port=3000)
```

### Serve static files

```python
import os
import mimetypes
from bunpy.serve import serve

STATIC_DIR = "./public"

def handler(req):
    if req.method != "GET":
        return {"status": 405, "body": "method not allowed"}

    # Strip the leading slash and resolve against the static dir
    rel_path = req.path.lstrip("/") or "index.html"
    abs_path = os.path.join(STATIC_DIR, rel_path)

    # Prevent path traversal
    if not os.path.abspath(abs_path).startswith(os.path.abspath(STATIC_DIR)):
        return {"status": 403, "body": "forbidden"}

    if not os.path.isfile(abs_path):
        return {"status": 404, "body": "not found"}

    content_type, _ = mimetypes.guess_type(abs_path)
    with open(abs_path, "rb") as f:
        return {
            "status": 200,
            "headers": {"Content-Type": content_type or "application/octet-stream"},
            "body": f.read(),
        }

serve(handler, port=8080)
```

### HTTPS / TLS

```python
from bunpy.serve import serve

def handler(req):
    return {"status": 200, "body": "secure!"}

serve(handler, port=443, tls={
    "cert": "/etc/ssl/certs/server.pem",
    "key": "/etc/ssl/private/server.key",
})
```

### Async handler

Handlers can be `async def`. bunpy runs async handlers on the event loop without any extra setup:

```python
import asyncio
import json
from bunpy.serve import serve

async def handler(req):
    # Simulate an async database call
    await asyncio.sleep(0.01)
    resp = await fetch("https://api.example.com/data")
    data = resp.json()
    return {
        "status": 200,
        "headers": {"Content-Type": "application/json"},
        "body": json.dumps(data),
    }

serve(handler, port=3000)
```

## Global shortcut

The `Bun.serve` global is an alias for `bunpy.serve.serve`:

```python
def handler(req):
    return "Hello!"

Bun.serve({"fetch": handler, "port": 3000})
```

This form accepts a single dict rather than positional arguments, matching Bun's API.
