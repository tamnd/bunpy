---
title: Database access with SQLAlchemy
description: Define models, run CRUD operations, manage relationships, and run Alembic migrations with SQLAlchemy and SQLite.
---

## Install

```bash
bunpy add sqlalchemy alembic
```

## Connect and create tables

SQLAlchemy's `DeclarativeBase` is the starting point for every ORM project. Define your models once, then call `create_all` to materialize the schema:

```python
from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase

engine = create_engine("sqlite:///blog.db", echo=False)

class Base(DeclarativeBase):
    pass
```

## Define models

```python
from datetime import datetime
from typing import Optional

from sqlalchemy import String, Text, ForeignKey, DateTime, func
from sqlalchemy.orm import Mapped, mapped_column, relationship

from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase

engine = create_engine("sqlite:///blog.db")

class Base(DeclarativeBase):
    pass

class User(Base):
    __tablename__ = "users"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    email: Mapped[str] = mapped_column(String(128), unique=True, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=func.now())

    posts: Mapped[list["Post"]] = relationship("Post", back_populates="author", cascade="all, delete-orphan")

    def __repr__(self) -> str:
        return f"<User id={self.id} username={self.username!r}>"


class Post(Base):
    __tablename__ = "posts"

    id: Mapped[int] = mapped_column(primary_key=True)
    title: Mapped[str] = mapped_column(String(256), nullable=False)
    body: Mapped[str] = mapped_column(Text, nullable=False)
    published: Mapped[bool] = mapped_column(default=False)
    author_id: Mapped[int] = mapped_column(ForeignKey("users.id"), nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=func.now())

    author: Mapped["User"] = relationship("User", back_populates="posts")

    def __repr__(self) -> str:
        return f"<Post id={self.id} title={self.title!r}>"


Base.metadata.create_all(engine)
```

## CRUD operations

All database work happens inside a `Session`. Use it as a context manager so commits and rollbacks are handled automatically:

```python
from sqlalchemy.orm import Session

# --- Create ---
with Session(engine) as session:
    alice = User(username="alice", email="alice@example.com")
    bob = User(username="bob", email="bob@example.com")
    session.add_all([alice, bob])
    session.flush()   # assigns alice.id / bob.id

    post1 = Post(title="Hello SQLAlchemy", body="Getting started with the ORM.", author_id=alice.id)
    post2 = Post(title="Async SQLAlchemy", body="Using asyncio with SQLAlchemy 2.", author_id=alice.id, published=True)
    session.add_all([post1, post2])
    session.commit()
    print(f"Created user {alice.id}, posts {post1.id}, {post2.id}")
```

```python
from sqlalchemy import select
from sqlalchemy.orm import Session

# --- Read ---
with Session(engine) as session:
    # fetch by primary key
    user = session.get(User, 1)
    print(user)

    # query with filter
    stmt = select(User).where(User.username == "alice")
    alice = session.scalars(stmt).one()
    print(alice.email)

    # all posts by alice, ordered by newest first
    posts_stmt = (
        select(Post)
        .where(Post.author_id == alice.id)
        .order_by(Post.created_at.desc())
    )
    posts = session.scalars(posts_stmt).all()
    for post in posts:
        print(post.title, "-", post.published)
```

```python
from sqlalchemy.orm import Session

# --- Update ---
with Session(engine) as session:
    post = session.get(Post, 1)
    if post:
        post.published = True
        post.title = "Hello SQLAlchemy (updated)"
        session.commit()
        print("Updated:", post.title)
```

```python
from sqlalchemy.orm import Session

# --- Delete ---
with Session(engine) as session:
    post = session.get(Post, 2)
    if post:
        session.delete(post)
        session.commit()
        print("Deleted post 2")
```

## Relationships and eager loading

By default SQLAlchemy uses lazy loading - it fires a second query when you access `user.posts`. For most use cases, eager loading is cleaner:

```python
from sqlalchemy import select
from sqlalchemy.orm import Session, selectinload

with Session(engine) as session:
    stmt = (
        select(User)
        .options(selectinload(User.posts))
        .where(User.username == "alice")
    )
    alice = session.scalars(stmt).one()

    for post in alice.posts:
        print(f"  [{post.id}] {post.title} - published={post.published}")
```

`selectinload` emits one query per relationship, which is usually better than `joinedload` for one-to-many when you expect many rows.

## Filter, order, and paginate

```python
from sqlalchemy import select
from sqlalchemy.orm import Session

def list_published_posts(session: Session, page: int = 1, per_page: int = 10) -> list[Post]:
    stmt = (
        select(Post)
        .where(Post.published == True)
        .order_by(Post.created_at.desc())
        .offset((page - 1) * per_page)
        .limit(per_page)
    )
    return list(session.scalars(stmt))

with Session(engine) as session:
    posts = list_published_posts(session, page=1, per_page=5)
    for p in posts:
        print(p.id, p.title)
```

## Count and aggregate

```python
from sqlalchemy import select, func
from sqlalchemy.orm import Session

with Session(engine) as session:
    total = session.scalar(select(func.count()).select_from(Post).where(Post.published == True))
    print(f"Published posts: {total}")

    # posts per user
    stmt = (
        select(User.username, func.count(Post.id).label("post_count"))
        .join(Post, Post.author_id == User.id, isouter=True)
        .group_by(User.id)
        .order_by(func.count(Post.id).desc())
    )
    with Session(engine) as session:
        for row in session.execute(stmt):
            print(row.username, row.post_count)
```

## Full working example: users and posts API

```python
from __future__ import annotations

from datetime import datetime
from typing import Optional

from sqlalchemy import (
    DateTime, ForeignKey, String, Text, create_engine, func, select
)
from sqlalchemy.orm import (
    DeclarativeBase, Mapped, Session, mapped_column, relationship, selectinload
)


engine = create_engine("sqlite:///blog.db", echo=False)


class Base(DeclarativeBase):
    pass


class User(Base):
    __tablename__ = "users"
    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(64), unique=True)
    email: Mapped[str] = mapped_column(String(128), unique=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=func.now())
    posts: Mapped[list[Post]] = relationship(back_populates="author", cascade="all, delete-orphan")


class Post(Base):
    __tablename__ = "posts"
    id: Mapped[int] = mapped_column(primary_key=True)
    title: Mapped[str] = mapped_column(String(256))
    body: Mapped[str] = mapped_column(Text)
    published: Mapped[bool] = mapped_column(default=False)
    author_id: Mapped[int] = mapped_column(ForeignKey("users.id"))
    created_at: Mapped[datetime] = mapped_column(DateTime, default=func.now())
    author: Mapped[User] = relationship(back_populates="posts")


Base.metadata.create_all(engine)


def create_user(username: str, email: str) -> User:
    with Session(engine) as session:
        user = User(username=username, email=email)
        session.add(user)
        session.commit()
        session.refresh(user)
        return user


def create_post(author_id: int, title: str, body: str, published: bool = False) -> Post:
    with Session(engine) as session:
        post = Post(author_id=author_id, title=title, body=body, published=published)
        session.add(post)
        session.commit()
        session.refresh(post)
        return post


def get_user_with_posts(user_id: int) -> Optional[User]:
    with Session(engine) as session:
        return session.scalars(
            select(User).options(selectinload(User.posts)).where(User.id == user_id)
        ).one_or_none()


if __name__ == "__main__":
    alice = create_user("alice", "alice@example.com")
    create_post(alice.id, "First post", "Hello world!", published=True)
    create_post(alice.id, "Draft", "Work in progress.")

    user = get_user_with_posts(alice.id)
    if user:
        print(f"{user.username} has {len(user.posts)} posts:")
        for p in user.posts:
            print(f"  - {p.title} (published={p.published})")
```

## Alembic migrations

Alembic tracks schema changes as versioned migration scripts. Initialize it once, then generate and apply migrations as your models evolve.

**Initialize Alembic:**

```bash
bunpy run alembic init migrations
```

Edit `alembic.ini` - set the database URL:

```
sqlalchemy.url = sqlite:///blog.db
```

Edit `migrations/env.py` - import your Base so Alembic can detect model changes:

```python
# migrations/env.py (relevant section)
from myapp.models import Base   # import your DeclarativeBase
target_metadata = Base.metadata
```

**Generate a migration:**

```bash
bunpy run alembic revision --autogenerate -m "add users and posts tables"
```

**Apply the migration:**

```bash
bunpy run alembic upgrade head
```

**Add a column later:**

Add `bio: Mapped[Optional[str]] = mapped_column(Text, nullable=True)` to `User`, then:

```bash
bunpy run alembic revision --autogenerate -m "add user bio"
bunpy run alembic upgrade head
```

**Roll back:**

```bash
bunpy run alembic downgrade -1
```

## Run the example

```bash
bunpy blog.py
```

SQLAlchemy 2.x with `Mapped` annotations gives you full type-checker support - mypy and pyright understand the column types without any plugins.
