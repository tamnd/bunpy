---
title: Deploy to Fly.io
description: Deploy a bunpy app to Fly.io with fly.toml, secrets, SQLite volumes, health checks, and scale-to-zero.
---

Fly.io runs your app in a VM close to your users, supports SQLite with persistent volumes, and lets you scale to zero when there is no traffic. This guide covers a complete deployment including secrets, health checks, and the full `fly.toml` configuration.

---

## Prerequisites

- A bunpy project with `pyproject.toml` and a committed `uv.lock`
- The `flyctl` CLI: `curl -L https://fly.io/install.sh | sh`
- A Fly.io account: `fly auth signup` or `fly auth login`

---

## Step 1: Launch

From the project root:

```bash
fly launch
```

Fly will ask a series of questions:

- **App name** — pick something globally unique, e.g. `myapp-prod`
- **Region** — choose the region closest to your users or database
- **Postgres** — say no for now if you plan to use SQLite
- **Redis** — say no unless your app needs it
- **Deploy now** — say no; you'll configure `fly.toml` first

`fly launch` writes a `fly.toml` file to the current directory. You'll edit it in the next step.

---

## Step 2: fly.toml

Replace the generated `fly.toml` with the following. Adjust `app`, `primary_region`, and the internal port to match your app.

```toml
app = "myapp-prod"
primary_region = "sin"

[build]
  # Fly builds using the Dockerfile in the project root.
  # Remove this section if you want Fly to auto-detect (it will use nixpacks).

[env]
  PORT = "8080"
  LOG_LEVEL = "info"

[[services]]
  protocol = "tcp"
  internal_port = 8080
  auto_stop_machines = true     # scale to zero when idle
  auto_start_machines = true    # wake on incoming request
  min_machines_running = 0      # allows full scale-to-zero

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [services.concurrency]
    type = "connections"
    hard_limit = 25
    soft_limit = 20

  [[services.http_checks]]
    interval = "10s"
    timeout = "5s"
    grace_period = "15s"
    method = "GET"
    path = "/healthz"

[[vm]]
  memory = "512mb"
  cpu_kind = "shared"
  cpus = 1
```

### Build section

If your project has a `Dockerfile`, Fly uses it. The Docker guide in this documentation covers the multi-stage Dockerfile that works well with Fly. If you omit the `[build]` section entirely, Fly falls back to nixpacks, which detects `pyproject.toml` and runs Python.

For a nixpacks-based deploy without Docker, add a build command:

```toml
[build]
  builder = "nixpacks"
  build-args = { BUILD_CMD = "bunpy install --frozen" }
```

---

## Step 3: Dockerfile for Fly

The recommended approach is a multi-stage Dockerfile. Fly builds it on their infrastructure and deploys the resulting image.

```dockerfile
FROM ubuntu:24.04 AS builder

RUN apt-get update && apt-get install -y curl ca-certificates --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
ENV PATH="/root/.bunpy/bin:$PATH"

WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen --target /app/site-packages
COPY src/ src/

FROM python:3.14-slim
WORKDIR /app
ENV PYTHONPATH=/app/site-packages
COPY --from=builder /app/site-packages /app/site-packages
COPY --from=builder /app/src /app/src

RUN useradd -m appuser
USER appuser

EXPOSE 8080
CMD ["python", "-m", "myapp"]
```

---

## Step 4: Secrets

Secrets are environment variables that Fly encrypts at rest and injects at runtime. Never put secrets in `fly.toml` — it is committed to source control.

```bash
# Set a secret
fly secrets set SECRET_KEY=supersecret

# Set multiple secrets at once
fly secrets set \
  DATABASE_URL=postgresql://user:pass@host/db \
  REDIS_URL=redis://localhost:6379 \
  SENDGRID_API_KEY=SG.abc123

# List secrets (values are redacted)
fly secrets list

# Remove a secret
fly secrets unset SECRET_KEY
```

Secrets set with `fly secrets set` trigger a rolling restart. Your app sees them as regular environment variables:

```python
import os
secret_key = os.environ["SECRET_KEY"]
database_url = os.environ.get("DATABASE_URL")
```

---

## Step 5: SQLite volumes

If your app uses SQLite, you need a persistent volume so data survives restarts and deployments.

```bash
# Create a 1 GB volume in the primary region
fly volumes create myapp_data --size 1 --region sin
```

Mount the volume in `fly.toml`:

```toml
[[mounts]]
  source = "myapp_data"
  destination = "/data"
```

Then point SQLite at the mounted path:

```python
import os
DB_PATH = os.environ.get("DB_PATH", "/data/app.db")
```

For production SQLite on Fly, consider using [LiteFS](https://fly.io/docs/litefs/) for replication across multiple VMs. A single-VM SQLite setup is simpler and works well for apps with moderate write volume.

---

## Step 6: Deploy

```bash
fly deploy
```

Fly builds the Docker image, pushes it to their registry, and starts a rolling deployment. New VMs start and pass health checks before old VMs are stopped.

Watch the deployment:

```bash
fly deploy --now
```

---

## Logs

```bash
# Stream live logs
fly logs

# Logs for a specific instance
fly logs --instance <instance-id>

# Show recent logs without streaming
fly logs --no-tail
```

Example output:

```
2026-04-28T10:23:01Z app[abc123] sin [info] Starting: bunpy server.py
2026-04-28T10:23:02Z app[abc123] sin [info] Listening on 0.0.0.0:8080
2026-04-28T10:23:12Z app[abc123] sin [info] GET /healthz 200 1ms
```

---

## Scale to zero

The `fly.toml` above sets `auto_stop_machines = true` and `min_machines_running = 0`. When no requests arrive for a configurable idle period (default 5 minutes), Fly stops the VM and bills you nothing. The next request wakes it up in ~2 seconds.

Scale-to-zero is appropriate for development environments, low-traffic services, and staging. For production with latency requirements, set `min_machines_running = 1`.

```bash
# Manual scaling
fly scale count 2           # run 2 VMs at all times
fly scale count 0           # stop all VMs (same as scale-to-zero)

# Show current VM status
fly status
```

---

## Health checks

The `[[services.http_checks]]` block in `fly.toml` pings `/healthz` every 10 seconds. Fly marks the VM unhealthy if the endpoint returns a non-2xx status or does not respond within the timeout.

Add the endpoint to your app:

```python
# FastAPI
from fastapi import FastAPI
app = FastAPI()

@app.get("/healthz")
def health():
    return {"status": "ok"}
```

```python
# Plain Python HTTP server
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
        else:
            self.send_response(404)
            self.end_headers()
```

During deployment, the `grace_period = "15s"` setting gives your app time to start before health checks begin. Increase this if your app takes longer to initialize (e.g., it runs database migrations on startup).

---

## Full deployment checklist

- `fly.toml` is committed to source control (it has no secrets)
- `uv.lock` is committed and up to date
- Secrets are set with `fly secrets set`, not in `fly.toml`
- Health check endpoint responds with 200 before the grace period ends
- Volume is created before first deploy if the app uses SQLite
- `fly deploy` passes and the app appears in `fly status`
