---
title: Deploy to Render
description: Deploy a bunpy app to Render using render.yaml — build command, start command, managed Postgres, env vars, and health checks.
---

Render is a fully managed cloud platform with a straightforward YAML configuration. You define your service, database, environment variables, and health checks in a single `render.yaml` file committed to your repository. This guide walks through a complete setup.

---

## Prerequisites

- A bunpy project with `pyproject.toml` and a committed `uv.lock`
- A Render account at [render.com](https://render.com)
- The repository pushed to GitHub or GitLab (Render deploys from Git)

---

## render.yaml

Create `render.yaml` in the repository root. This is the Infrastructure as Code file that Render reads when you connect the repository.

```yaml
services:
  - type: web
    name: myapp
    runtime: python
    buildCommand: |
      curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
      export PATH="$HOME/.bunpy/bin:$PATH"
      bunpy install --frozen
    startCommand: bunpy server.py
    healthCheckPath: /healthz
    envVars:
      - key: PORT
        value: "10000"
      - key: LOG_LEVEL
        value: info
      - key: DATABASE_URL
        fromDatabase:
          name: myapp-db
          property: connectionString
      - key: SECRET_KEY
        generateValue: true
    autoDeploy: true
    plan: starter

databases:
  - name: myapp-db
    databaseName: myapp
    user: myapp
    plan: starter
```

### Key fields

**`buildCommand`** runs once during each deploy. This is where you install bunpy and run `bunpy install --frozen`. Render caches the build environment between deploys, but the cache is keyed on the build command output. If `uv.lock` changes, the entire install runs again.

**`startCommand`** is what Render runs to start the web process. Render injects `PORT` into the environment; your app must listen on that port.

**`healthCheckPath`** — Render pings this endpoint after deploy. The service is considered healthy when it returns HTTP 200. Render routes traffic to the new instance only after the health check passes. If it fails within the timeout, Render rolls back to the previous deploy automatically.

**`autoDeploy: true`** — triggers a new deploy every time you push to the connected branch.

---

## Environment variables

There are three ways to set environment variables in `render.yaml`:

**Static value:**

```yaml
- key: LOG_LEVEL
  value: info
```

**Generated value (Render creates a random secret):**

```yaml
- key: SECRET_KEY
  generateValue: true
```

Render generates this once and reuses it across deploys. It is visible in the dashboard but not in source control.

**From a database:**

```yaml
- key: DATABASE_URL
  fromDatabase:
    name: myapp-db
    property: connectionString
```

Render injects the connection string from the managed Postgres instance. The available `property` values are `connectionString`, `host`, `port`, `user`, `password`, and `database`.

**Sensitive variables in the dashboard:**

For secrets that cannot be in `render.yaml` (API keys, third-party credentials), add them in the Render dashboard under Environment > Environment Variables. These are not stored in source control.

---

## Managed PostgreSQL

The `databases` block in `render.yaml` provisions a managed Postgres instance:

```yaml
databases:
  - name: myapp-db
    databaseName: myapp
    user: myapp
    plan: starter
```

Render handles backups, patching, and connection pooling. The `starter` plan includes 1 GB storage and is free during development. Use `standard` or `pro` for production.

Connect from Python:

```python
import os
from sqlalchemy import create_engine

engine = create_engine(os.environ["DATABASE_URL"])
```

The `DATABASE_URL` is injected via the `fromDatabase` reference in `render.yaml`, so you do not need to configure it manually.

---

## Health check endpoint

Render requires the health check endpoint to respond within 30 seconds of startup. Add `/healthz` to your app:

```python
# FastAPI
from fastapi import FastAPI
app = FastAPI()

@app.get("/healthz")
def health():
    return {"status": "ok"}
```

```python
# Django
# urls.py
from django.http import JsonResponse
from django.urls import path

def health(request):
    return JsonResponse({"status": "ok"})

urlpatterns = [
    path("healthz", health),
    # ... other routes
]
```

If your app runs database migrations on startup, the health check should wait until migrations complete before returning 200. A simple approach:

```python
import os
from sqlalchemy import text

@app.get("/healthz")
def health():
    try:
        with engine.connect() as conn:
            conn.execute(text("SELECT 1"))
        return {"status": "ok"}
    except Exception as e:
        return {"status": "error", "detail": str(e)}, 503
```

This verifies the database is reachable, not just that the HTTP server started.

---

## Autoscaling

Render's autoscaling is configured in the dashboard or via `render.yaml` on paid plans:

```yaml
services:
  - type: web
    name: myapp
    # ... other config
    scaling:
      minInstances: 1
      maxInstances: 5
      targetMemoryPercent: 75
      targetCPUPercent: 70
```

Render scales up when CPU or memory exceeds the target and scales down when load drops. The `minInstances: 1` setting prevents scale-to-zero, which is appropriate for services that need low-latency responses.

For development or low-traffic services, set `minInstances: 0` (requires `plan: starter` or higher on the new billing model).

---

## Build caching

Render caches build artifacts between deploys. To take advantage of this, structure `buildCommand` so that the expensive steps (installing bunpy, running `bunpy install --frozen`) only re-run when necessary.

Render does not support arbitrary cache keys like GitHub Actions does. The cache is invalidated when the build command changes or when you manually clear it in the dashboard. In practice, `bunpy install --frozen` is fast enough on warm cache (~1–2 seconds for a resolved lockfile) that build caching matters less here.

---

## Deploy process

After committing `render.yaml` and pushing:

1. Go to [dashboard.render.com](https://dashboard.render.com)
2. Click "New" > "Blueprint"
3. Select your repository
4. Render reads `render.yaml` and shows a preview of the resources it will create
5. Confirm, and Render provisions the database and deploys the web service

Subsequent deploys happen automatically on push (with `autoDeploy: true`) or manually via the dashboard.

Watch deploy logs in the dashboard or via the Render API:

```bash
# Using the Render API with curl
curl -H "Authorization: Bearer $RENDER_API_KEY" \
  "https://api.render.com/v1/services/$SERVICE_ID/deploys?limit=5"
```

---

## Full project structure

```
myapp/
  pyproject.toml
  uv.lock
  render.yaml
  src/
    myapp/
      __init__.py
      server.py
```

The `render.yaml` file is the single source of truth for the infrastructure. Committing it to source control means anyone on the team can reproduce the exact same Render setup in a new account by connecting the repository.
