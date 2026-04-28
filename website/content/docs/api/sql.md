---
title: bunpy.sql
description: SQLite database interface.
---

```python
import bunpy.sql as sql
```

## Opening a database

```python
db = sql.open("mydb.sqlite")
db = sql.open(":memory:")   # in-memory database
```

## Executing queries

```python
# DDL
db.exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")

# Insert with positional parameters
db.exec("INSERT INTO users (name) VALUES (?)", "Alice")

# Query - returns list of row dicts
rows = db.query("SELECT * FROM users")
for row in rows:
    print(row["name"])

# Single row
user = db.queryOne("SELECT * FROM users WHERE id = ?", 1)

# Scalar value
count = db.queryValue("SELECT COUNT(*) FROM users")
```

## Transactions

```python
with db.transaction():
    db.exec("INSERT INTO users (name) VALUES (?)", "Bob")
    db.exec("INSERT INTO users (name) VALUES (?)", "Carol")
# committed on exit; rolled back on exception
```

## Prepared statements

```python
stmt = db.prepare("INSERT INTO users (name) VALUES (?)")
for name in ["Dave", "Eve", "Frank"]:
    stmt.run(name)
```

## Closing

```python
db.close()
```
