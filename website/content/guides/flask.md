---
title: Build a web app with Flask
description: Build a full Flask web app with routes, Jinja2 templates, forms, SQLite, static files, and a gunicorn production setup.
---

## Create the project

```bash
bunpy create --template minimal my-flask-app
cd my-flask-app
bunpy add flask gunicorn
```

## Project layout

```
my-flask-app/
  app.py
  pyproject.toml
  uv.lock
  templates/
    base.html
    index.html
    about.html
    contact.html
  static/
    style.css
  instance/
    app.db        ← created automatically at runtime
```

## Application factory

`app.py` — the entire application in one file for clarity. A larger project would split this into a package with blueprints.

```python
import os
import sqlite3
from flask import (
    Flask,
    render_template,
    request,
    redirect,
    url_for,
    jsonify,
    g,
    abort,
)

app = Flask(__name__)
app.secret_key = os.environ.get("SECRET_KEY", "dev-only-secret-change-in-prod")

DATABASE = os.environ.get("DATABASE_URL", "instance/app.db")
```

## Database helpers

Flask's `g` object gives you a per-request database connection that is opened lazily and closed when the request tears down.

```python
def get_db() -> sqlite3.Connection:
    if "db" not in g:
        os.makedirs("instance", exist_ok=True)
        g.db = sqlite3.connect(DATABASE, detect_types=sqlite3.PARSE_DECLTYPES)
        g.db.row_factory = sqlite3.Row
    return g.db


@app.teardown_appcontext
def close_db(error):
    db = g.pop("db", None)
    if db is not None:
        db.close()


def init_db():
    db = get_db()
    db.execute(
        """
        CREATE TABLE IF NOT EXISTS messages (
            id      INTEGER PRIMARY KEY AUTOINCREMENT,
            name    TEXT NOT NULL,
            email   TEXT NOT NULL,
            body    TEXT NOT NULL,
            created TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """
    )
    db.commit()


@app.before_request
def ensure_schema():
    init_db()
```

## Routes

### Index and about

```python
@app.get("/")
def index():
    db = get_db()
    messages = db.execute(
        "SELECT id, name, body, created FROM messages ORDER BY id DESC LIMIT 5"
    ).fetchall()
    return render_template("index.html", messages=messages)


@app.get("/about")
def about():
    return render_template("about.html")
```

### JSON data endpoint

```python
@app.get("/api/data")
def api_data():
    db = get_db()
    rows = db.execute("SELECT id, name, body, created FROM messages").fetchall()
    return jsonify([dict(row) for row in rows])
```

### Contact form (GET + POST)

```python
@app.route("/contact", methods=["GET", "POST"])
def contact():
    errors: dict[str, str] = {}

    if request.method == "POST":
        name = request.form.get("name", "").strip()
        email = request.form.get("email", "").strip()
        body = request.form.get("body", "").strip()

        if not name:
            errors["name"] = "Name is required."
        if not email or "@" not in email:
            errors["email"] = "A valid email is required."
        if not body:
            errors["body"] = "Message cannot be empty."

        if not errors:
            db = get_db()
            db.execute(
                "INSERT INTO messages (name, email, body) VALUES (?, ?, ?)",
                (name, email, body),
            )
            db.commit()
            return redirect(url_for("index"))

        return render_template(
            "contact.html",
            errors=errors,
            form={"name": name, "email": email, "body": body},
        )

    return render_template("contact.html", errors={}, form={})
```

## Jinja2 templates

### `templates/base.html`

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{% block title %}My Flask App{% endblock %}</title>
  <link rel="stylesheet" href="{{ url_for('static', filename='style.css') }}">
</head>
<body>
  <nav>
    <a href="{{ url_for('index') }}">Home</a>
    <a href="{{ url_for('about') }}">About</a>
    <a href="{{ url_for('contact') }}">Contact</a>
  </nav>
  <main>
    {% block content %}{% endblock %}
  </main>
</body>
</html>
```

### `templates/index.html`

```html
{% extends "base.html" %}
{% block title %}Home{% endblock %}
{% block content %}
<h1>Recent messages</h1>
{% if messages %}
  <ul>
    {% for msg in messages %}
      <li><strong>{{ msg.name }}</strong> — {{ msg.body }} <small>({{ msg.created }})</small></li>
    {% endfor %}
  </ul>
{% else %}
  <p>No messages yet. <a href="{{ url_for('contact') }}">Send one!</a></p>
{% endif %}
{% endblock %}
```

### `templates/about.html`

```html
{% extends "base.html" %}
{% block title %}About{% endblock %}
{% block content %}
<h1>About</h1>
<p>This app is built with Flask and managed by bunpy.</p>
{% endblock %}
```

### `templates/contact.html`

```html
{% extends "base.html" %}
{% block title %}Contact{% endblock %}
{% block content %}
<h1>Contact</h1>
<form method="post">
  <label>Name
    <input type="text" name="name" value="{{ form.get('name', '') }}">
    {% if errors.name %}<span class="error">{{ errors.name }}</span>{% endif %}
  </label>
  <label>Email
    <input type="email" name="email" value="{{ form.get('email', '') }}">
    {% if errors.email %}<span class="error">{{ errors.email }}</span>{% endif %}
  </label>
  <label>Message
    <textarea name="body">{{ form.get('body', '') }}</textarea>
    {% if errors.body %}<span class="error">{{ errors.body }}</span>{% endif %}
  </label>
  <button type="submit">Send</button>
</form>
{% endblock %}
```

## Static files

### `static/style.css`

```css
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

body { font-family: system-ui, sans-serif; max-width: 720px; margin: 0 auto; padding: 1rem; }
nav  { display: flex; gap: 1rem; padding: 0.5rem 0 1rem; border-bottom: 1px solid #e5e7eb; }
nav a { text-decoration: none; color: #3b82f6; }
nav a:hover { text-decoration: underline; }
main { padding-top: 1.5rem; }
h1   { margin-bottom: 1rem; }
ul   { list-style: none; display: flex; flex-direction: column; gap: 0.5rem; }
label { display: flex; flex-direction: column; gap: 0.25rem; margin-bottom: 1rem; }
input, textarea { padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 4px; }
textarea { min-height: 120px; }
button { padding: 0.5rem 1.5rem; background: #3b82f6; color: #fff; border: none; border-radius: 4px; cursor: pointer; }
button:hover { background: #2563eb; }
.error { color: #ef4444; font-size: 0.875rem; }
```

## Run in development

```bash
bunpy app.py
# * Running on http://127.0.0.1:5000
```

Set the environment variable `FLASK_DEBUG=1` to enable the reloader and debugger:

```bash
FLASK_DEBUG=1 bunpy app.py
```

## Configuration with environment variables

Flask's `app.secret_key` is already pulled from `SECRET_KEY`. Extend the pattern for any setting:

```python
app.config.update(
    DATABASE=os.environ.get("DATABASE_URL", "instance/app.db"),
    MAIL_SERVER=os.environ.get("MAIL_SERVER", "localhost"),
    DEBUG=os.environ.get("FLASK_DEBUG", "0") == "1",
)
```

Store secrets in a `.env` file and load them before starting the server:

```bash
# .env
SECRET_KEY=change-me-in-production
DATABASE_URL=instance/prod.db
```

```bash
bunpy --env-file .env app.py
```

## Run with gunicorn in production

```bash
bunpy add gunicorn
gunicorn "app:app" --workers 4 --bind 0.0.0.0:8000
```

gunicorn forks worker processes so each gets its own SQLite connection through `get_db`. For write-heavy workloads, swap SQLite for PostgreSQL and use a connection pool.

A minimal `Procfile` for platforms like Render or Railway:

```
web: gunicorn "app:app" --workers 4 --bind 0.0.0.0:$PORT
```

## Docker deployment

```dockerfile
FROM python:3.12-slim

WORKDIR /app

RUN pip install bunpy --no-cache-dir

COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen

COPY . .

EXPOSE 8000
CMD ["gunicorn", "app:app", "--workers", "4", "--bind", "0.0.0.0:8000"]
```

```bash
docker build -t my-flask-app .
docker run -p 8000:8000 -e SECRET_KEY=prod-secret my-flask-app
```

## What to add next

- **Flask-Login**: session-based auth with `login_required` decorators.
- **Blueprints**: split routes into `auth.py`, `admin.py`, and `api.py` as the app grows.
- **WTForms**: replace the manual form validation with declarative form classes and CSRF protection.
- **SQLAlchemy**: swap the raw `sqlite3` calls for an ORM that handles migrations with Alembic.
