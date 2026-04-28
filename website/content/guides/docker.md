---
title: Docker deployment
description: Build a minimal Docker image with a compiled bunpy binary.
---

## Dockerfile — minimal scratch image

```dockerfile
# Build stage
FROM ubuntu:24.04 AS builder

RUN apt-get update && apt-get install -y curl
RUN curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash

WORKDIR /app
COPY pyproject.toml .
COPY src/ src/

ENV PATH="/root/.bunpy/bin:$PATH"
RUN bunpy install --frozen
RUN bunpy build --compile src/myapp/__main__.py -o /app/myapp

# Run stage — scratch or distroless
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/myapp /myapp
ENTRYPOINT ["/myapp"]
```

Build and run:

```bash
docker build -t myapp .
docker run --rm -p 3000:3000 myapp
```

## .dockerignore

```
.bunpy/
__pycache__/
*.pyc
.env
```

## Multi-platform build

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t myapp:latest --push .
```

## Passing environment variables

```bash
docker run --rm \
  -e DATABASE_URL=sqlite:///data/app.db \
  -e PORT=8080 \
  -v "$(pwd)/data:/data" \
  myapp
```

## Size comparison

| Base image | Approx size |
|------------|-------------|
| `ubuntu:24.04` | ~80 MB |
| `distroless/static` | ~10 MB |
| `scratch` | ~8 MB |

A compiled bunpy binary is self-contained and runs on scratch with no libc
dependency (CGO is disabled; goipy is pure Go).
