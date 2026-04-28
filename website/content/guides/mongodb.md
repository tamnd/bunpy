---
title: MongoDB with bunpy
description: Connect to MongoDB with PyMongo, run CRUD operations, build aggregation pipelines, create indexes, and handle errors in bunpy applications.
weight: 15
---

## Install

```bash
bunpy add pymongo
```

Make sure MongoDB is reachable. For local development:

```bash
# Docker — quickest option
docker run -d -p 27017:27017 --name mongo mongo:7

# macOS with Homebrew
brew install mongodb-community && brew services start mongodb-community
```

## Connect

```python
from pymongo import MongoClient
from pymongo.database import Database

client = MongoClient("mongodb://localhost:27017/")
db: Database = client["myapp"]

# verify the connection
info = client.server_info()
print(f"Connected — MongoDB {info['version']}")
```

For Atlas or any remote cluster, pass the full connection string. Store it in an environment variable rather than in source:

```python
import os
from pymongo import MongoClient

MONGO_URL = os.environ.get("MONGO_URL", "mongodb://localhost:27017/")
client = MongoClient(MONGO_URL)
db = client["myapp"]
```

A `MongoClient` manages a connection pool internally. Create one instance at startup and share it across your application — do not create a new client per request.

## Insert documents

`insert_one` inserts a single document. MongoDB adds an `_id` field (a `bson.ObjectId`) if you do not provide one:

```python
from datetime import datetime, timezone
from pymongo import MongoClient

client = MongoClient("mongodb://localhost:27017/")
db = client["myapp"]
users = db["users"]

result = users.insert_one({
    "username": "alice",
    "email": "alice@example.com",
    "role": "admin",
    "created_at": datetime.now(timezone.utc),
})

print(f"Inserted: {result.inserted_id}")
```

`insert_many` inserts a list in one round trip:

```python
from datetime import datetime, timezone

new_users = [
    {"username": "bob",   "email": "bob@example.com",   "role": "editor"},
    {"username": "carol", "email": "carol@example.com", "role": "viewer"},
    {"username": "dave",  "email": "dave@example.com",  "role": "editor"},
]

result = users.insert_many(new_users)
print(f"Inserted {len(result.inserted_ids)} documents")
```

Documents are Python dicts. Any serializable value — strings, numbers, lists, nested dicts, datetimes, booleans — is stored as BSON natively. You do not define a schema upfront.

## Find documents

`find_one` returns the first matching document or `None`. `find` returns a cursor you can iterate:

```python
# single document by field
user = users.find_one({"username": "alice"})
print(user["email"])

# all documents matching a filter
editors = list(users.find({"role": "editor"}))
for u in editors:
    print(u["username"])

# projection: include only certain fields (1 = include, 0 = exclude)
usernames = list(users.find({}, {"_id": 0, "username": 1, "role": 1}))
print(usernames)

# sort, skip, limit
recent = list(
    users.find({})
         .sort("created_at", -1)
         .skip(0)
         .limit(10)
)
```

`find` returns a lazy cursor — no data is fetched until you iterate it. Call `list()` to materialize all results, or iterate directly to process one document at a time without loading everything into memory.

## Update documents

`update_one` modifies the first matching document. `update_many` modifies all matches. Use `$set` to change specific fields without overwriting the whole document:

```python
from pymongo import ReturnDocument

# update a single field
result = users.update_one(
    {"username": "alice"},
    {"$set": {"role": "superadmin", "verified": True}},
)
print(f"Matched: {result.matched_count}, Modified: {result.modified_count}")

# upsert: insert if not found
users.update_one(
    {"username": "eve"},
    {"$set": {"email": "eve@example.com", "role": "viewer"}},
    upsert=True,
)

# increment a numeric field
db["page_views"].update_one(
    {"page": "/home"},
    {"$inc": {"views": 1}},
    upsert=True,
)

# find_one_and_update: return the document after modification
updated = users.find_one_and_update(
    {"username": "bob"},
    {"$set": {"role": "admin"}},
    return_document=ReturnDocument.AFTER,
)
print(updated["role"])   # admin
```

Common update operators: `$set` (set fields), `$unset` (remove fields), `$inc` (increment), `$push` (append to array), `$pull` (remove from array), `$addToSet` (append if not present).

## Delete documents

```python
# delete the first match
result = users.delete_one({"username": "dave"})
print(f"Deleted: {result.deleted_count}")

# delete all matches
result = users.delete_many({"role": "viewer"})
print(f"Deleted {result.deleted_count} viewers")

# delete the entire collection
db["temp_data"].drop()
```

`delete_one` is safer than `delete_many` when you mean to remove a specific record — it will not cascade if your filter is accidentally broad.

## Aggregation pipeline

The aggregation pipeline transforms documents through a sequence of stages. It runs entirely on the server, making it far more efficient than fetching documents and processing them in Python:

```python
from datetime import datetime, timezone
from pymongo import MongoClient

client = MongoClient("mongodb://localhost:27017/")
db = client["myapp"]
orders = db["orders"]

# seed some data
orders.insert_many([
    {"customer": "alice", "product": "laptop",  "amount": 1200.0, "region": "west"},
    {"customer": "alice", "product": "mouse",   "amount":   25.0, "region": "west"},
    {"customer": "bob",   "product": "monitor", "amount":  450.0, "region": "east"},
    {"customer": "bob",   "product": "laptop",  "amount": 1100.0, "region": "east"},
    {"customer": "carol", "product": "keyboard","amount":   80.0, "region": "west"},
    {"customer": "carol", "product": "mouse",   "amount":   25.0, "region": "east"},
])

# revenue by region, sorted descending
pipeline = [
    {"$group": {
        "_id": "$region",
        "total_revenue": {"$sum": "$amount"},
        "order_count":   {"$sum": 1},
        "avg_order":     {"$avg": "$amount"},
    }},
    {"$sort": {"total_revenue": -1}},
    {"$project": {
        "_id": 0,
        "region":        "$_id",
        "total_revenue": {"$round": ["$total_revenue", 2]},
        "order_count":   1,
        "avg_order":     {"$round": ["$avg_order", 2]},
    }},
]

for row in orders.aggregate(pipeline):
    print(row)
```

Common pipeline stages:

| Stage | Purpose |
|---|---|
| `$match` | Filter documents (like `find`) |
| `$group` | Group and aggregate |
| `$sort` | Order results |
| `$project` | Reshape fields |
| `$lookup` | Join from another collection |
| `$unwind` | Flatten arrays |
| `$limit` / `$skip` | Paginate |
| `$count` | Count matching documents |

Add a `$match` stage early to filter before grouping — MongoDB can use an index on `$match` but not on later stages.

## Indexes for performance

Without an index, every query does a full collection scan. Add indexes on fields you filter or sort by:

```python
from pymongo import MongoClient, ASCENDING, DESCENDING, TEXT

client = MongoClient("mongodb://localhost:27017/")
db = client["myapp"]
users = db["users"]
posts = db["posts"]

# unique index — enforce uniqueness and speed up lookups
users.create_index("email", unique=True)

# compound index — efficient for queries that filter on both fields
posts.create_index([("author_id", ASCENDING), ("created_at", DESCENDING)])

# text index — enables full-text search with $text
posts.create_index([("title", TEXT), ("body", TEXT)])

# list all indexes on a collection
for idx in users.list_indexes():
    print(idx["name"], idx.get("unique", False))
```

Use `explain()` to check whether a query uses an index:

```python
plan = users.find({"email": "alice@example.com"}).explain()
print(plan["queryPlanner"]["winningPlan"]["stage"])   # IXSCAN = index used
```

Drop an index you no longer need:

```python
users.drop_index("email_1")
```

## Error handling

Wrap database operations in `try/except` to handle connection failures, duplicate key violations, and timeouts:

```python
from pymongo import MongoClient
from pymongo.errors import (
    ConnectionFailure,
    DuplicateKeyError,
    OperationFailure,
    ServerSelectionTimeoutError,
)

client = MongoClient("mongodb://localhost:27017/", serverSelectionTimeoutMS=3000)
db = client["myapp"]
users = db["users"]
users.create_index("email", unique=True)

def create_user(username: str, email: str) -> dict | None:
    try:
        result = users.insert_one({"username": username, "email": email})
        return {"id": str(result.inserted_id), "username": username}
    except DuplicateKeyError:
        print(f"User with email {email!r} already exists.")
        return None
    except ServerSelectionTimeoutError:
        print("Could not reach MongoDB — check the connection URL.")
        raise
    except OperationFailure as exc:
        print(f"Database operation failed: {exc.details}")
        raise

user = create_user("alice", "alice@example.com")
print(user)

duplicate = create_user("alice2", "alice@example.com")
print(duplicate)   # None — duplicate key, handled gracefully
```

`serverSelectionTimeoutMS` controls how long PyMongo waits before raising `ServerSelectionTimeoutError` when no server is available. Set it low in health-check code so failures surface quickly.

## Full working example

A small product catalogue using all the operations above:

```python
import os
from datetime import datetime, timezone
from pymongo import MongoClient, ASCENDING
from pymongo.errors import DuplicateKeyError

MONGO_URL = os.environ.get("MONGO_URL", "mongodb://localhost:27017/")
client = MongoClient(MONGO_URL)
db = client["catalogue"]
products = db["products"]

products.create_index("sku", unique=True)
products.create_index([("category", ASCENDING), ("price", ASCENDING)])

def add_product(sku: str, name: str, price: float, category: str) -> str | None:
    try:
        result = products.insert_one({
            "sku": sku, "name": name, "price": price,
            "category": category,
            "created_at": datetime.now(timezone.utc),
        })
        return str(result.inserted_id)
    except DuplicateKeyError:
        print(f"SKU {sku!r} already exists.")
        return None

def update_price(sku: str, new_price: float) -> bool:
    result = products.update_one({"sku": sku}, {"$set": {"price": new_price}})
    return result.modified_count == 1

def list_by_category(category: str, max_price: float | None = None) -> list[dict]:
    query: dict = {"category": category}
    if max_price is not None:
        query["price"] = {"$lte": max_price}
    return list(products.find(query, {"_id": 0}).sort("price", ASCENDING))

def category_summary() -> list[dict]:
    pipeline = [
        {"$group": {"_id": "$category", "count": {"$sum": 1}, "avg_price": {"$avg": "$price"}}},
        {"$sort": {"count": -1}},
        {"$project": {"_id": 0, "category": "$_id", "count": 1, "avg_price": {"$round": ["$avg_price", 2]}}},
    ]
    return list(products.aggregate(pipeline))

if __name__ == "__main__":
    add_product("LAP-001", "Laptop Pro 15",  1299.99, "laptops")
    add_product("LAP-002", "Laptop Air 13",   899.00, "laptops")
    add_product("MON-001", "4K Monitor 27",   549.00, "monitors")
    add_product("KBD-001", "Mechanical KB",    89.99, "accessories")
    add_product("MSE-001", "Wireless Mouse",   39.99, "accessories")

    update_price("LAP-002", 849.00)

    print("Laptops under $1000:")
    for p in list_by_category("laptops", max_price=1000):
        print(f"  {p['sku']}  {p['name']}  ${p['price']}")

    print("\nCategory summary:")
    for row in category_summary():
        print(f"  {row['category']}: {row['count']} items, avg ${row['avg_price']}")
```

## Run the example

```bash
bunpy catalogue.py
```

PyMongo is synchronous. For async web frameworks like FastAPI, use `motor` (`bunpy add motor`) — it wraps PyMongo with an async interface and the same query API. Connection strings, CRUD methods, and aggregation pipelines are identical; only the `await` calls differ.
