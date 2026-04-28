---
title: Deploy to Railway
description: Step-by-step guide to deploying a bunpy app on Railway with PostgreSQL, environment variables, and health checks.
---

Railway is a good fit for bunpy projects. It handles build and run automatically, exposes environment variables cleanly, and provisions a managed PostgreSQL instance with one command. This guide walks through a complete deployment from a fresh repository to a live service.


## Prerequisites

- A bunpy project with `pyproject.toml` and a committed `uv.lock`
- A GitHub account with the repository pushed
- The Railway CLI installed: `npm install -g @railway/cli`
- A Railway account (free tier works)


## Step 1: Create the project

```bash
# Log in
railway login

# Create a new project in the current directory
railway init

# When prompted, choose "Empty Project" and give it a name
# Railway will link the current directory to the project
```

If you prefer the dashboard, go to [railway.app](https://railway.app), click "New Project", then "Deploy from GitHub repo" and select your repository. Come back to the CLI for the rest.


## Step 2: Procfile

Railway uses a `Procfile` to know how to run your app. Create one in the repository root:

```
web: bunpy server.py
```

For a project with multiple process types:

```
web: bunpy server.py
worker: bunpy worker.py
```

The `web` process type gets an assigned `PORT` environment variable and is connected to Railway's routing. Read `PORT` in your app:

```python
import os
port = int(os.environ.get("PORT", 8080))
```


## Step 3: railway.toml

Create `railway.toml` in the repository root to tell Railway how to build and run the service:

```toml
[build]
builder = "nixpacks"
buildCommand = "bunpy install --frozen"

[deploy]
startCommand = "bunpy server.py"
restartPolicyType = "on-failure"
restartPolicyMaxRetries = 3
healthcheckPath = "/healthz"
healthcheckTimeout = 10
```

### nixpacks detection

Railway uses nixpacks to detect the project type. nixpacks looks for `pyproject.toml` and installs Python automatically. The `buildCommand` runs after the base environment is set up. `bunpy install --frozen` reads `uv.lock` and installs your exact dependency tree.

If nixpacks cannot find bunpy in `PATH`, add an install step:

```toml
[build]
builder = "nixpacks"
buildCommand = "curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash && bunpy install --frozen"
```

Or pin it with a `nixpacks.toml`:

```toml
[phases.setup]
nixPkgs = ["python314", "curl"]

[phases.install]
cmds = [
  "curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash",
  "bunpy install --frozen"
]
```


## Step 4: Health check endpoint

Add a `/healthz` route to your app. Railway pings this to decide whether the deploy succeeded. A simple example with the `http` module from the standard library:

```python
from http.server import BaseHTTPRequestHandler, HTTPServer
import os

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
        else:
            # route to your actual handler
            ...

port = int(os.environ.get("PORT", 8080))
HTTPServer(("0.0.0.0", port), Handler).serve_forever()
```

For FastAPI:

```python
from fastapi import FastAPI
import os

app = FastAPI()

@app.get("/healthz")
def health():
    return {"status": "ok"}
```


## Step 5: Add PostgreSQL

```bash
railway add --plugin postgresql
```

This provisions a managed Postgres instance and injects the following environment variables into your service automatically:

- `DATABASE_URL` - full connection string
- `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`

Read the connection string in your app:

```python
import os
DATABASE_URL = os.environ["DATABASE_URL"]
```

Railway's Postgres addon uses the same connection string format as most Python ORMs. With SQLAlchemy:

```python
from sqlalchemy import create_engine
engine = create_engine(os.environ["DATABASE_URL"])
```


## Step 6: Environment variables

Set environment variables in the Railway dashboard under Settings > Variables, or via the CLI:

```bash
# Set a single variable
railway variables set SECRET_KEY=supersecret

# Set multiple variables from a .env file (does not commit the file)
railway variables set --from-file .env.production
```

Variables set in the dashboard are injected at runtime. They are not stored in your repository.

For local development, Railway can inject the remote variables into your local shell:

```bash
railway run bunpy server.py
```

This pulls all project variables and runs the command with them in the environment, so your local app connects to the same database as production. Useful for debugging production-specific issues without hardcoding connection strings.


## Step 7: Deploy

```bash
# Deploy the current branch
railway up

# Deploy and follow logs in real time
railway up --detach && railway logs --follow
```

Railway triggers a build, runs `buildCommand`, then starts your process with `startCommand`. The build log shows each step. If the build fails, the previous deployment stays active.


## Deploy logs

```bash
# Stream live logs from the running service
railway logs --follow

# Show logs for a specific service if you have multiple
railway logs --service web --follow
```

Example output:

```
[build] Installing bunpy...
[build] bunpy install --frozen
[build] Resolved 47 packages in 0.3s
[build] Installed 47 packages in 1.2s
[deploy] Starting: bunpy server.py
[deploy] Listening on 0.0.0.0:3456
```


## Environment-specific config

Use Railway environments to separate staging and production. Create a staging environment in the dashboard, then deploy to it:

```bash
railway environment staging
railway up
```

Each environment has its own variable set and its own Postgres instance. The staging environment uses the same `railway.toml` build config but different secrets.


## Full project structure

```
myapp/
  pyproject.toml
  uv.lock
  Procfile
  railway.toml
  nixpacks.toml      # optional, for custom Nix packages
  src/
    myapp/
      __init__.py
      server.py
```

The `Procfile` and `railway.toml` together tell Railway everything it needs. There is no platform-specific runtime file to maintain beyond these two.
