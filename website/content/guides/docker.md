---
title: Deploy with Docker
description: Two approaches to containerizing a bunpy app - full install with layer caching, and a single .pyz file in a scratch image.
---

Docker and bunpy fit together well. You get deterministic builds from `uv.lock`, fast rebuilds from layer caching, and either a full Python environment or a self-contained binary depending on what your app needs.

This guide covers two approaches:

1. **Full install** - copies your dependencies into the image, runs with `python:3.14-slim`. Good for apps that use C extensions or need the full Python stdlib.
2. **.pyz bundle** - packages everything into a single archive, runs on a minimal base. Good for pure-Python services.


## Approach 1: Full install with layer caching

The key insight for fast Docker rebuilds is to copy `pyproject.toml` and `uv.lock` before copying your source code. Docker caches each layer. If neither lockfile changes between builds, the `bunpy install` layer is served from cache and the entire dependency installation step takes under a second.

```dockerfile
# syntax=docker/dockerfile:1
FROM ubuntu:24.04 AS builder

RUN apt-get update && apt-get install -y curl ca-certificates --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
ENV PATH="/root/.bunpy/bin:$PATH"

WORKDIR /app

# Copy lockfiles first - this layer is cached until the lockfiles change
COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen --target /app/site-packages

# Now copy source - cache miss here does not invalidate the install layer above
COPY src/ src/

# ──────────────────────────────────────────────
FROM python:3.14-slim AS runtime

WORKDIR /app
ENV PYTHONPATH=/app/site-packages

COPY --from=builder /app/site-packages /app/site-packages
COPY --from=builder /app/src /app/src

# Non-root user
RUN useradd -m appuser
USER appuser

EXPOSE 8080
CMD ["python", "-m", "myapp"]
```

Build and run:

```bash
docker build -t myapp:latest .
docker run --rm -p 8080:8080 \
  -e DATABASE_URL=postgresql://user:pass@db/mydb \
  myapp:latest
```

### What `--frozen` does

`bunpy install --frozen` fails if `uv.lock` is not up to date with `pyproject.toml`. In CI and Docker builds you always want this flag. It catches the case where a developer added a dependency to `pyproject.toml` but forgot to run `bunpy pm lock` and commit the updated lockfile. Without `--frozen`, the build would silently install a different set of packages than what is checked into source control.

### What `--target` does

`--target /app/site-packages` installs packages into a specific directory instead of the system site-packages. This makes it straightforward to copy the installed dependencies as a single directory between build stages, and it means the runtime image does not need bunpy installed at all.


## Approach 2: .pyz bundle in a scratch image

For pure-Python services the `.pyz` approach produces the smallest possible image. A `.pyz` archive is a zipapp: a single file that contains your source code and all dependencies. Python's zipimport machinery handles the rest.

```dockerfile
# syntax=docker/dockerfile:1
FROM ubuntu:24.04 AS builder

RUN apt-get update && apt-get install -y curl ca-certificates --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
ENV PATH="/root/.bunpy/bin:$PATH"

WORKDIR /app

COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen

COPY src/ src/

# Build the .pyz archive
RUN bunpy build src/myapp/__main__.py -o /app/myapp.pyz

# ──────────────────────────────────────────────
# Runtime stage: python:3.14-slim is enough for a .pyz
# Use scratch only if you compile to a native binary with --compile
FROM python:3.14-slim AS runtime

COPY --from=builder /app/myapp.pyz /myapp.pyz

RUN useradd -m appuser
USER appuser

EXPOSE 8080
ENTRYPOINT ["python", "/myapp.pyz"]
```

If your app has no C extension dependencies and you use `bunpy build --compile`, you can go all the way to a `scratch` or `distroless/static` base:

```dockerfile
FROM gcr.io/distroless/static-debian12 AS runtime
COPY --from=builder /app/myapp /myapp
ENTRYPOINT ["/myapp"]
```

The compiled binary is statically linked. It includes the Go runtime and the goipy interpreter. There is nothing else to install.


## ARM64 cross-compile

Build images for both `linux/amd64` and `linux/arm64` in one command with buildx:

```bash
# One-time setup
docker buildx create --use --name multiarch
docker buildx inspect --bootstrap

# Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag myregistry/myapp:latest \
  --push \
  .
```

For the native binary approach, set `GOARCH` in the build stage:

```dockerfile
FROM ubuntu:24.04 AS builder
ARG TARGETARCH
ENV GOARCH=$TARGETARCH
# bunpy build will cross-compile using the GOARCH env var
RUN bunpy build --compile src/myapp/__main__.py -o /app/myapp
```

Docker passes `TARGETARCH` automatically when building multi-platform. bunpy reads `GOARCH` and produces the correct binary.


## .dockerignore

Always include a `.dockerignore` to keep the build context small and prevent secrets from leaking into the image:

```
.bunpy/
__pycache__/
*.pyc
*.pyo
.env
.env.*
.git/
.pytest_cache/
dist/
*.pyz
.venv/
node_modules/
```

The `.bunpy/` directory contains your local cache. It can be hundreds of megabytes. Without `.dockerignore`, Docker sends the entire working directory to the build daemon for each build.


## Image size comparison

| Approach | Base image | Approx size |
|---|---|---|
| Full install | `python:3.14-slim` | ~180 MB |
| .pyz archive | `python:3.14-slim` | ~160 MB |
| Compiled binary | `distroless/static-debian12` | ~25 MB |
| Compiled binary | `scratch` | ~22 MB |

The compiled binary route gives the smallest image by a large margin because it carries no OS packages and no Python installation. The tradeoff is that C extension dependencies (`psycopg`, `cryptography`, and similar) do not work in a compiled binary - they require a CPython runtime.


## Passing environment variables

```bash
docker run --rm \
  -e DATABASE_URL=postgresql://user:pass@db/mydb \
  -e PORT=8080 \
  -e LOG_LEVEL=info \
  -v "$(pwd)/data:/data" \
  myapp:latest
```

For production, prefer Docker secrets or your orchestration platform's secret management over `-e` flags.


## Docker Compose example

```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgresql://appuser:secret@db/appdb
      PORT: "8080"
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16
    environment:
      POSTGRES_USER: appuser
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: appdb
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U appuser"]
      interval: 5s
      timeout: 3s
      retries: 5
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

Run with `docker compose up --build`. The `depends_on` condition ensures the app container does not start until Postgres passes its health check.
