---
title: bunpy.serve
description: HTTP server built-in to bunpy.
---

```python
from bunpy.serve import serve
```

## Basic server

```python
from bunpy.serve import serve

def handler(req):
    return {"status": 200, "body": "Hello, world!"}

serve(handler, port=3000)
print("Listening on http://localhost:3000")
```

## Request object

| Attribute | Type | Description |
|-----------|------|-------------|
| `req.method` | str | HTTP method (`GET`, `POST`, ...) |
| `req.url` | str | Full request URL |
| `req.path` | str | URL path |
| `req.query` | dict | Parsed query parameters |
| `req.headers` | dict | Request headers |
| `req.body` | bytes | Raw request body |
| `req.json()` | dict | Parse body as JSON |
| `req.text()` | str | Decode body as UTF-8 |

## Response

Return a dict with optional keys:

```python
return {
    "status": 200,
    "headers": {"Content-Type": "application/json"},
    "body": '{"ok": true}',
}
```

Or return a string (defaults to `200 text/plain`):

```python
return "Hello!"
```

## Options

```python
serve(handler, port=3000, hostname="0.0.0.0")
```

| Option | Default | Description |
|--------|---------|-------------|
| `port` | 3000 | TCP port |
| `hostname` | `"127.0.0.1"` | Bind address |
| `development` | False | Enable request logging |
