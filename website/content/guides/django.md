---
title: Build a Django app with bunpy
description: Scaffold a Django project, define models, wire up views and URLs, run migrations, and deploy to production with bunpy.
---

## Install Django

```bash
bunpy create --template minimal my-django-app
cd my-django-app
bunpy add django gunicorn
```

## Bootstrap the project

Django ships with a management command runner. Use `bunpy run` to invoke it without activating a virtual environment manually.

```bash
bunpy run -m django startproject mysite .
bunpy run manage.py startapp blog
```

Your layout now looks like this:

```
my-django-app/
  manage.py
  mysite/
    __init__.py
    settings.py
    urls.py
    wsgi.py
  blog/
    __init__.py
    admin.py
    apps.py
    models.py
    views.py
    urls.py          ← create this file
    migrations/
      __init__.py
  pyproject.toml
  uv.lock
```

## Settings

Open `mysite/settings.py` and make three changes.

Register the app:

```python
INSTALLED_APPS = [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
    "blog",          # add this
]
```

Keep SQLite for development (Django's default), and pull the secret key from the environment:

```python
import os

SECRET_KEY = os.environ.get("DJANGO_SECRET_KEY", "dev-only-change-me")
DEBUG = os.environ.get("DEBUG", "1") == "1"
ALLOWED_HOSTS = os.environ.get("ALLOWED_HOSTS", "localhost 127.0.0.1").split()
```

Static and media files:

```python
STATIC_URL = "/static/"
STATIC_ROOT = BASE_DIR / "staticfiles"
```

## Define the model

Edit `blog/models.py`:

```python
from django.db import models
from django.utils import timezone


class Post(models.Model):
    title = models.CharField(max_length=200)
    slug = models.SlugField(unique=True)
    body = models.TextField()
    author = models.ForeignKey(
        "auth.User",
        on_delete=models.CASCADE,
        related_name="posts",
    )
    published_at = models.DateTimeField(null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["-created_at"]

    def __str__(self) -> str:
        return self.title

    def publish(self) -> None:
        self.published_at = timezone.now()
        self.save(update_fields=["published_at"])

    @property
    def is_published(self) -> bool:
        return self.published_at is not None
```

## Create and run migrations

```bash
bunpy run manage.py makemigrations blog
bunpy run manage.py migrate
```

The output confirms Django created the `blog_post` table in `db.sqlite3`.

## Register with the admin

Edit `blog/admin.py`:

```python
from django.contrib import admin
from .models import Post


@admin.register(Post)
class PostAdmin(admin.ModelAdmin):
    list_display = ["title", "author", "is_published", "created_at"]
    list_filter = ["published_at"]
    search_fields = ["title", "body"]
    prepopulated_fields = {"slug": ("title",)}
    date_hierarchy = "created_at"
    readonly_fields = ["created_at", "updated_at"]

    actions = ["publish_posts"]

    @admin.action(description="Publish selected posts")
    def publish_posts(self, request, queryset):
        for post in queryset:
            post.publish()
        self.message_user(request, f"{queryset.count()} post(s) published.")
```

Create a superuser so you can log in:

```bash
bunpy run manage.py createsuperuser
```

## Write the views

Edit `blog/views.py`:

```python
from django.shortcuts import render, get_object_or_404
from django.http import HttpRequest, HttpResponse
from django.core.paginator import Paginator
from .models import Post


def post_list(request: HttpRequest) -> HttpResponse:
    posts = Post.objects.filter(published_at__isnull=False).select_related("author")
    paginator = Paginator(posts, per_page=10)
    page_obj = paginator.get_page(request.GET.get("page"))
    return render(request, "blog/post_list.html", {"page_obj": page_obj})


def post_detail(request: HttpRequest, slug: str) -> HttpResponse:
    post = get_object_or_404(Post, slug=slug, published_at__isnull=False)
    return render(request, "blog/post_detail.html", {"post": post})
```

## Wire up URLs

Create `blog/urls.py`:

```python
from django.urls import path
from . import views

app_name = "blog"

urlpatterns = [
    path("", views.post_list, name="post_list"),
    path("<slug:slug>/", views.post_detail, name="post_detail"),
]
```

Include it in `mysite/urls.py`:

```python
from django.contrib import admin
from django.urls import path, include

urlpatterns = [
    path("admin/", admin.site.urls),
    path("blog/", include("blog.urls")),
]
```

## Templates

Create `blog/templates/blog/base.html`:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{% block title %}My Blog{% endblock %}</title>
</head>
<body>
  <header><a href="{% url 'blog:post_list' %}">My Blog</a></header>
  <main>{% block content %}{% endblock %}</main>
</body>
</html>
```

Create `blog/templates/blog/post_list.html`:

```html
{% extends "blog/base.html" %}
{% block title %}Posts{% endblock %}
{% block content %}
<h1>Posts</h1>
{% for post in page_obj %}
  <article>
    <h2><a href="{% url 'blog:post_detail' post.slug %}">{{ post.title }}</a></h2>
    <p>By {{ post.author.get_full_name|default:post.author.username }} &mdash; {{ post.published_at|date:"N j, Y" }}</p>
    <p>{{ post.body|truncatewords:30 }}</p>
  </article>
{% empty %}
  <p>No posts yet.</p>
{% endfor %}

{% if page_obj.has_other_pages %}
  <nav>
    {% if page_obj.has_previous %}<a href="?page={{ page_obj.previous_page_number }}">Prev</a>{% endif %}
    Page {{ page_obj.number }} of {{ page_obj.paginator.num_pages }}
    {% if page_obj.has_next %}<a href="?page={{ page_obj.next_page_number }}">Next</a>{% endif %}
  </nav>
{% endif %}
{% endblock %}
```

Create `blog/templates/blog/post_detail.html`:

```html
{% extends "blog/base.html" %}
{% block title %}{{ post.title }}{% endblock %}
{% block content %}
<article>
  <h1>{{ post.title }}</h1>
  <p>By {{ post.author.get_full_name|default:post.author.username }} &mdash; {{ post.published_at|date:"N j, Y" }}</p>
  {{ post.body|linebreaks }}
</article>
<a href="{% url 'blog:post_list' %}">&larr; All posts</a>
{% endblock %}
```

## Run the development server

```bash
bunpy run manage.py runserver
# Django version 5.x, using settings 'mysite.settings'
# Starting development server at http://127.0.0.1:8000/
```

Visit `http://127.0.0.1:8000/admin/` to create a Post through the admin UI, then open `http://127.0.0.1:8000/blog/` to see the public list.

## Collect static files for production

```bash
bunpy run manage.py collectstatic --noinput
```

## Deploy to production with gunicorn

```bash
DEBUG=0 \
DJANGO_SECRET_KEY=your-production-key \
ALLOWED_HOSTS=example.com \
gunicorn mysite.wsgi:application --workers 4 --bind 0.0.0.0:8000
```

## Docker deployment

```dockerfile
FROM python:3.12-slim

WORKDIR /app

RUN pip install bunpy --no-cache-dir

COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen

COPY . .

RUN bunpy run manage.py collectstatic --noinput

EXPOSE 8000
CMD ["gunicorn", "mysite.wsgi:application", "--workers", "4", "--bind", "0.0.0.0:8000"]
```

Set environment variables at container runtime rather than baking them into the image:

```bash
docker build -t my-django-app .
docker run -p 8000:8000 \
  -e DJANGO_SECRET_KEY=prod-secret \
  -e DEBUG=0 \
  -e ALLOWED_HOSTS=example.com \
  my-django-app
```

Run migrations before the first deploy:

```bash
docker run --rm \
  -e DJANGO_SECRET_KEY=prod-secret \
  -e DEBUG=0 \
  my-django-app \
  bunpy run manage.py migrate
```

## What to add next

- **Django REST Framework**: add `djangorestframework` and expose `/api/posts/` with serializers and viewsets.
- **PostgreSQL**: change `DATABASES["default"]["ENGINE"]` to `django.db.backends.postgresql` and add `psycopg2-binary`.
- **Celery**: wire up async email sending or image processing with a Redis broker (see the background tasks guide).
- **django-environ**: replace the manual `os.environ.get` calls with a typed `.env` loader.
