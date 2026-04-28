---
title: Build a REST API with FastAPI
description: Create a production-ready CRUD API with FastAPI, Pydantic models, dependency injection, async tests, and a .pyz bundle.
---

## Create the project

```bash
bunpy create --template minimal my-api
cd my-api
bunpy add fastapi uvicorn httpx pytest pytest-asyncio
```

Your `pyproject.toml` now lists those dependencies. Run `bunpy install` any time you clone the project on a new machine.

## Project layout

```
my-api/
  server.py
  pyproject.toml
  uv.lock
  tests/
    test_items.py
```

## Define the models

FastAPI relies on Pydantic for request and response validation. Create `server.py`:

```python
from __future__ import annotations

from typing import Optional
from fastapi import FastAPI, HTTPException, Depends
from pydantic import BaseModel

app = FastAPI(title="Items API", version="1.0.0")


# ---------------------------------------------------------------------------
# Pydantic models
# ---------------------------------------------------------------------------

class Item(BaseModel):
    id: int
    name: str
    description: Optional[str] = None
    price: float
    in_stock: bool = True


class CreateItemRequest(BaseModel):
    name: str
    description: Optional[str] = None
    price: float
    in_stock: bool = True


class UpdateItemRequest(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    price: Optional[float] = None
    in_stock: Optional[bool] = None
```

## In-memory database and dependency injection

A real app would use SQLAlchemy or another ORM. Here we use a plain dict so the example runs without a database server, but the `get_db` dependency pattern is identical to what you would use in production.

```python
# ---------------------------------------------------------------------------
# Fake database
# ---------------------------------------------------------------------------

_db: dict[int, Item] = {
    1: Item(id=1, name="Laptop", price=999.00, description="14-inch, 16 GB RAM"),
    2: Item(id=2, name="Mouse", price=29.99),
}
_next_id = 3


def get_db() -> dict[int, Item]:
    """Yield the shared in-memory store. Swap this for a real Session later."""
    return _db
```

## CRUD routes

```python
# ---------------------------------------------------------------------------
# Routes
# ---------------------------------------------------------------------------

@app.get("/items", response_model=list[Item])
def list_items(db: dict[int, Item] = Depends(get_db)):
    return list(db.values())


@app.post("/items", response_model=Item, status_code=201)
def create_item(body: CreateItemRequest, db: dict[int, Item] = Depends(get_db)):
    global _next_id
    item = Item(id=_next_id, **body.model_dump())
    db[_next_id] = item
    _next_id += 1
    return item


@app.get("/items/{item_id}", response_model=Item)
def get_item(item_id: int, db: dict[int, Item] = Depends(get_db)):
    item = db.get(item_id)
    if not item:
        raise HTTPException(status_code=404, detail=f"Item {item_id} not found")
    return item


@app.patch("/items/{item_id}", response_model=Item)
def update_item(
    item_id: int,
    body: UpdateItemRequest,
    db: dict[int, Item] = Depends(get_db),
):
    item = db.get(item_id)
    if not item:
        raise HTTPException(status_code=404, detail=f"Item {item_id} not found")
    updated = item.model_copy(update=body.model_dump(exclude_unset=True))
    db[item_id] = updated
    return updated


@app.delete("/items/{item_id}", status_code=204)
def delete_item(item_id: int, db: dict[int, Item] = Depends(get_db)):
    if item_id not in db:
        raise HTTPException(status_code=404, detail=f"Item {item_id} not found")
    del db[item_id]
```

## Entry point

```python
# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

Run the server:

```bash
bunpy server.py
# Listening on http://0.0.0.0:8000
```

Or use the short form:

```bash
bunpy run server.py
```

Test the endpoints manually:

```bash
# List all items
curl http://localhost:8000/items

# Create a new item
curl -X POST http://localhost:8000/items \
  -H "Content-Type: application/json" \
  -d '{"name": "Keyboard", "price": 79.99}'

# Get a single item
curl http://localhost:8000/items/3

# Partial update
curl -X PATCH http://localhost:8000/items/3 \
  -H "Content-Type: application/json" \
  -d '{"price": 69.99}'

# Delete
curl -X DELETE http://localhost:8000/items/3
```

FastAPI auto-generates interactive docs at `http://localhost:8000/docs`.

## Write async tests

Create `tests/test_items.py`. The `httpx.AsyncClient` mounts directly onto the FastAPI app, so no real port is opened.

```python
import pytest
import pytest_asyncio
import httpx
from server import app

pytestmark = pytest.mark.asyncio


@pytest_asyncio.fixture
async def client():
    async with httpx.AsyncClient(app=app, base_url="http://test") as c:
        yield c


async def test_list_items(client):
    r = await client.get("/items")
    assert r.status_code == 200
    items = r.json()
    assert len(items) >= 2


async def test_create_item(client):
    r = await client.post("/items", json={"name": "Headset", "price": 149.99})
    assert r.status_code == 201
    data = r.json()
    assert data["name"] == "Headset"
    assert data["id"] > 0


async def test_get_item(client):
    r = await client.get("/items/1")
    assert r.status_code == 200
    assert r.json()["name"] == "Laptop"


async def test_get_item_not_found(client):
    r = await client.get("/items/9999")
    assert r.status_code == 404


async def test_update_item(client):
    r = await client.patch("/items/2", json={"price": 24.99})
    assert r.status_code == 200
    assert r.json()["price"] == 24.99


async def test_delete_item(client):
    # Create first so we don't disturb other tests
    create = await client.post("/items", json={"name": "Temp", "price": 1.00})
    item_id = create.json()["id"]

    r = await client.delete(f"/items/{item_id}")
    assert r.status_code == 204

    r = await client.get(f"/items/{item_id}")
    assert r.status_code == 404
```

Run the test suite:

```bash
bunpy test
# Passed 6 tests in 0.42s
```

## Bundle to a .pyz executable

```bash
bunpy build server.py -o api.pyz
./api.pyz
# Listening on http://0.0.0.0:8000
```

The `.pyz` file is a self-contained ZIP-based archive. Copy it to any machine that has a compatible Python runtime and it runs without a `pip install` step.

## Run in Docker

```dockerfile
FROM python:3.12-slim

WORKDIR /app

# Install bunpy
RUN pip install bunpy --no-cache-dir

# Copy lockfile first so Docker layer cache is reused when only source changes
COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen

COPY server.py ./

EXPOSE 8000
CMD ["bunpy", "server.py"]
```

Build and run:

```bash
docker build -t my-api .
docker run -p 8000:8000 my-api
```

Or use the pre-compiled `.pyz` for an even smaller image — no bunpy needed at runtime:

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY api.pyz .
EXPOSE 8000
CMD ["python", "api.pyz"]
```

## What to add next

- **Database**: swap `get_db` for a `sqlalchemy.orm.Session` backed by PostgreSQL. The dependency injection pattern stays identical.
- **Auth**: add an `Authorization` header check inside a shared `Depends(verify_token)` dependency.
- **Pagination**: add `skip: int = 0` and `limit: int = 20` query parameters to `list_items`.
- **CORS**: `app.add_middleware(CORSMiddleware, allow_origins=["*"])` for browser clients.
