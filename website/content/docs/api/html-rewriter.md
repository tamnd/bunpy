---
title: bunpy.html_rewriter
description: HTML transformation API — rewrite HTML without a DOM.
---

```python
from bunpy.html_rewriter import HTMLRewriter
```

`HTMLRewriter` streams through HTML and calls registered handlers for matched
elements, comments, and text chunks.

## Basic usage

```python
from bunpy.html_rewriter import HTMLRewriter

output = []

class LinkHandler:
    def element(self, el):
        el.setAttribute("target", "_blank")
        output.append(el.getAttribute("href"))

html = '<a href="/page">link</a>'
result = HTMLRewriter().on("a", LinkHandler()).transform(html)
```

## Handler methods

| Method | Called when |
|--------|-------------|
| `element(el)` | An opening tag matches the selector |
| `text(chunk)` | A text node inside a matched element |
| `comments(comment)` | An HTML comment |

## Element API

| Method | Description |
|--------|-------------|
| `el.tagName` | Tag name (lowercase) |
| `el.getAttribute(name)` | Get attribute value |
| `el.setAttribute(name, value)` | Set attribute value |
| `el.removeAttribute(name)` | Remove an attribute |
| `el.prepend(html)` | Insert HTML before the element's content |
| `el.append(html)` | Insert HTML after the element's content |
| `el.replace(html)` | Replace the element with HTML |
| `el.remove()` | Remove the element and its children |

## CSS selectors

`HTMLRewriter` supports a subset of CSS selectors:

- `*` — any element
- `div`, `a`, `span` — tag name
- `#id` — by id attribute
- `.class` — by class attribute
- `[attr]` — has attribute
- `[attr="value"]` — attribute equals value
