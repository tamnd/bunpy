---
title: Data validation with Pydantic
description: Validate API payloads, load config from environment variables, generate JSON schemas, and model complex data with Pydantic v2.
---

## Install

```bash
bunpy add pydantic pydantic-settings
```

## Basic model

Define a model by subclassing `BaseModel`. Every annotated field is validated when you construct an instance:

```python
from pydantic import BaseModel

class User(BaseModel):
    id: int
    name: str
    email: str
    is_active: bool = True

user = User(id=1, name="Alice", email="alice@example.com")
print(user.id)        # 1
print(user.is_active) # True

# Pydantic coerces compatible types
user2 = User(id="42", name="Bob", email="bob@example.com")
print(type(user2.id))  # <class 'int'>
```

Invalid data raises `ValidationError` with field-level details:

```python
from pydantic import BaseModel, ValidationError

class User(BaseModel):
    id: int
    email: str

try:
    User(id="not-a-number", email="bad")
except ValidationError as exc:
    print(exc.error_count(), "error(s)")
    for error in exc.errors():
        print(error["loc"], error["msg"])
```

## Field validators

Use `@field_validator` for custom per-field logic:

```python
from pydantic import BaseModel, field_validator, EmailStr

class SignupRequest(BaseModel):
    username: str
    email: str
    password: str

    @field_validator("username")
    @classmethod
    def username_must_be_lowercase(cls, value: str) -> str:
        if value != value.lower():
            raise ValueError("username must be all lowercase")
        return value

    @field_validator("password")
    @classmethod
    def password_min_length(cls, value: str) -> str:
        if len(value) < 8:
            raise ValueError("password must be at least 8 characters")
        return value

req = SignupRequest(username="alice", email="alice@example.com", password="hunter42")
print(req.username)
```

## model_validator for cross-field validation

`@model_validator` runs after all fields are populated and lets you check relationships between them:

```python
from pydantic import BaseModel, model_validator
from datetime import date

class DateRange(BaseModel):
    start: date
    end: date

    @model_validator(mode="after")
    def end_must_be_after_start(self) -> "DateRange":
        if self.end <= self.start:
            raise ValueError("end date must be after start date")
        return self

r = DateRange(start=date(2024, 1, 1), end=date(2024, 12, 31))
print(r.start, "->", r.end)
```

## Nested models

Models compose naturally:

```python
from pydantic import BaseModel
from typing import Optional

class Address(BaseModel):
    street: str
    city: str
    country: str
    postal_code: Optional[str] = None

class Company(BaseModel):
    name: str
    address: Address
    employee_count: int

company = Company(
    name="Acme Corp",
    address={"street": "123 Main St", "city": "Springfield", "country": "US"},
    employee_count=250,
)
print(company.address.city)   # Springfield
```

## Serialization

Convert models to dicts and JSON with `model_dump` and `model_dump_json`:

```python
from pydantic import BaseModel
from datetime import datetime
from typing import Optional

class Post(BaseModel):
    id: int
    title: str
    body: str
    published_at: Optional[datetime] = None

post = Post(id=1, title="Hello", body="World", published_at=datetime.now())

# dict — by_alias and exclude_none are useful for API output
d = post.model_dump()
print(d)

# JSON string
j = post.model_dump_json(indent=2)
print(j)

# exclude sensitive fields
safe = post.model_dump(exclude={"body"})
print(safe)
```

Deserialize from a dict or JSON string:

```python
from pydantic import BaseModel
from datetime import datetime

class Post(BaseModel):
    id: int
    title: str
    published_at: datetime

# from dict
post = Post.model_validate({"id": 1, "title": "Hi", "published_at": "2024-06-01T12:00:00"})

# from JSON string
post2 = Post.model_validate_json('{"id":2,"title":"Bye","published_at":"2024-07-01T00:00:00"}')
print(post2.published_at.year)   # 2024
```

## JSON schema generation

Pydantic can emit a JSON Schema for your models — useful for documentation or validation on the client side:

```python
from pydantic import BaseModel, Field
from typing import Optional
import json

class CreateUserRequest(BaseModel):
    username: str = Field(min_length=3, max_length=32, description="Unique username")
    email: str = Field(description="User's email address")
    age: Optional[int] = Field(default=None, ge=0, le=150)

schema = CreateUserRequest.model_json_schema()
print(json.dumps(schema, indent=2))
```

Output:

```json
{
  "title": "CreateUserRequest",
  "type": "object",
  "properties": {
    "username": {"type": "string", "minLength": 3, "maxLength": 32},
    "email": {"type": "string"},
    "age": {"anyOf": [{"type": "integer"}, {"type": "null"}]}
  },
  "required": ["username", "email"]
}
```

## Settings from environment variables

`BaseSettings` reads values from environment variables (and `.env` files via `python-dotenv`):

```bash
bunpy add pydantic-settings python-dotenv
```

```python
from pydantic_settings import BaseSettings, SettingsConfigDict
from pydantic import Field

class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8")

    database_url: str = Field(description="Postgres connection string")
    secret_key: str = Field(min_length=32)
    debug: bool = False
    max_connections: int = 10
    allowed_hosts: list[str] = ["localhost"]

settings = Settings()
print(settings.database_url)
print(settings.debug)
```

Create a `.env` file:

```
DATABASE_URL=postgresql://user:pass@localhost/mydb
SECRET_KEY=supersecretkey1234567890abcdef1234
DEBUG=true
ALLOWED_HOSTS=["localhost","example.com"]
```

Run:

```bash
bunpy app.py
```

## Discriminated unions

When a field can hold one of several model types, use a `Literal` discriminator so Pydantic picks the right model at parse time:

```python
from pydantic import BaseModel
from typing import Literal, Union, Annotated
from pydantic import Field

class CreditCard(BaseModel):
    payment_type: Literal["credit_card"]
    card_number: str
    expiry: str
    cvv: str

class BankTransfer(BaseModel):
    payment_type: Literal["bank_transfer"]
    iban: str
    bic: str

class PayPal(BaseModel):
    payment_type: Literal["paypal"]
    email: str

PaymentMethod = Annotated[
    Union[CreditCard, BankTransfer, PayPal],
    Field(discriminator="payment_type"),
]

class Order(BaseModel):
    id: int
    amount: float
    payment: PaymentMethod

order = Order.model_validate({
    "id": 99,
    "amount": 49.99,
    "payment": {"payment_type": "paypal", "email": "buyer@example.com"},
})

print(type(order.payment).__name__)   # PayPal
print(order.payment.email)
```

## Real-world: API request and response models

A complete pattern for a FastAPI-style endpoint — even if you are not using FastAPI, defining explicit request/response shapes makes your code self-documenting and easy to test:

```python
from pydantic import BaseModel, field_validator, model_validator
from typing import Optional
from datetime import datetime

# --- Request models ---

class CreatePostRequest(BaseModel):
    title: str
    body: str
    tags: list[str] = []
    published: bool = False

    @field_validator("title")
    @classmethod
    def title_not_empty(cls, v: str) -> str:
        v = v.strip()
        if not v:
            raise ValueError("title cannot be blank")
        return v

    @field_validator("tags")
    @classmethod
    def normalize_tags(cls, tags: list[str]) -> list[str]:
        return [t.lower().strip() for t in tags if t.strip()]

class UpdatePostRequest(BaseModel):
    title: Optional[str] = None
    body: Optional[str] = None
    tags: Optional[list[str]] = None
    published: Optional[bool] = None

# --- Response models ---

class AuthorResponse(BaseModel):
    id: int
    username: str

class PostResponse(BaseModel):
    id: int
    title: str
    body: str
    tags: list[str]
    published: bool
    author: AuthorResponse
    created_at: datetime
    updated_at: datetime

# --- Simulate a handler ---

def create_post(request_body: dict, author_id: int) -> dict:
    req = CreatePostRequest.model_validate(request_body)

    post = PostResponse(
        id=1,
        title=req.title,
        body=req.body,
        tags=req.tags,
        published=req.published,
        author=AuthorResponse(id=author_id, username="alice"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    return post.model_dump(mode="json")

result = create_post(
    {"title": "  Hello World  ", "body": "Content here.", "tags": ["Python", "API"]},
    author_id=7,
)
print(result["title"])   # Hello World
print(result["tags"])    # ['python', 'api']
```

## Run the examples

```bash
bunpy validate.py
```

Pydantic v2 is built in Rust and is dramatically faster than v1. Use it as the canonical source of truth for your data shapes — validators, serialization, and schema generation all live in one place.
