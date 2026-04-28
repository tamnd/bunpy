---
title: bunpy.fetch
description: HTTP client — also available as a global.
---

`fetch` is available as a global in every bunpy script. It is also
importable from `bunpy.fetch` for explicit use.

```python
# as a global (no import needed)
resp = fetch("https://httpbin.org/get")

# or import explicitly
from bunpy.fetch import fetch
```

## Signature

```python
fetch(url, options={}) → Response
```

## Options

| Key | Type | Description |
|-----|------|-------------|
| `method` | str | HTTP method, default `"GET"` |
| `headers` | dict | Request headers |
| `body` | str or bytes | Request body |
| `redirect` | str | `"follow"` (default), `"manual"`, `"error"` |
| `timeout` | int | Timeout in milliseconds |

## Response

| Attribute / Method | Description |
|--------------------|-------------|
| `.status` | HTTP status code |
| `.status_code` | Alias for `.status` |
| `.ok` | `True` if `200 ≤ status < 300` |
| `.headers` | Response headers dict |
| `.text()` | Decode body as UTF-8 string |
| `.json()` | Parse body as JSON |
| `.bytes()` | Raw body bytes |
| `.url` | Final URL after redirects |

## Examples

```python
# GET
resp = fetch("https://api.github.com/users/octocat")
user = resp.json()
print(user["name"])

# POST JSON
resp = fetch("https://httpbin.org/post", {
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": '{"key": "value"}',
})
print(resp.status)   # 200

# With timeout
resp = fetch("https://slow.example.com", {"timeout": 5000})
```
