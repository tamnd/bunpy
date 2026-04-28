---
title: Build an HTTP server
description: Serve JSON APIs and static files with bunpy.serve.
---

## Hello world server

```python
from bunpy.serve import serve

def handler(req):
    return {"status": 200, "body": "Hello, world!"}

serve(handler, port=3000)
```

```bash
bunpy server.py
# Listening on http://localhost:3000
```

## JSON API

```python
from bunpy.serve import serve
import json

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
        body = req.json()
        new_user = {"id": len(users) + 1, "name": body["name"]}
        users.append(new_user)
        return {
            "status": 201,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps(new_user),
        }

    return {"status": 404, "body": "Not found"}

serve(handler, port=3000)
```

Test it:

```bash
curl http://localhost:3000/users
# [{"id": 1, "name": "Alice"}, ...]

curl -X POST http://localhost:3000/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Carol"}'
# {"id": 3, "name": "Carol"}
```

## Serve static files

```python
from bunpy.node import fs, path
from bunpy.serve import serve

def handler(req):
    file_path = path.join("public", req.path.lstrip("/"))
    if fs.existsSync(file_path) and fs.statSync(file_path).isFile():
        content = fs.readFileSync(file_path)
        return {"status": 200, "body": content}
    return {"status": 404, "body": "Not found"}

serve(handler, port=3000)
```

## Environment-based config

```python
import os
from bunpy.serve import serve

PORT = int(os.environ.get("PORT", "3000"))
HOST = os.environ.get("HOST", "127.0.0.1")

def handler(req):
    return "ok"

serve(handler, port=PORT, hostname=HOST)
```
