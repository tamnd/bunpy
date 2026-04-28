---
title: bunpy.semver - Semantic Versioning
description: Parse, compare, range-check, increment, and format semantic version strings in bunpy with full semver 2.0 support.
weight: 18
---

```python
import bunpy.semver as semver
```

`bunpy.semver` parses and compares version strings following the [Semantic Versioning 2.0.0](https://semver.org) spec. It handles pre-release identifiers, build metadata, range expressions, and version arithmetic used in dependency resolution and release tooling.

## Parsing versions

```python
import bunpy.semver as semver

v = semver.parse("1.2.3")
print(v.major)       # 1
print(v.minor)       # 2
print(v.patch)       # 3
print(v.prerelease)  # None
print(v.build)       # None

# Pre-release and build metadata
v2 = semver.parse("2.0.0-rc.1+build.42")
print(v2.prerelease)  # "rc.1"
print(v2.build)       # "build.42"
```

### semver.parse(version) → Version

Returns a `Version` object. Raises `ValueError` if the string is not valid semver.

| Version field | Type | Description |
|--------------|------|-------------|
| `major` | int | Major version |
| `minor` | int | Minor version |
| `patch` | int | Patch version |
| `prerelease` | str \| None | Pre-release identifier (e.g. `"rc.1"`) |
| `build` | str \| None | Build metadata (e.g. `"build.42"`) |

## Validation

```python
import bunpy.semver as semver

semver.valid("1.2.3")        # True
semver.valid("v1.2.3")       # True - leading "v" is accepted
semver.valid("1.2")          # False - missing patch
semver.valid("not-a-version") # False

# Coerce loose version strings
v = semver.coerce("1.2")      # → Version("1.2.0")
v = semver.coerce("v3")       # → Version("3.0.0")
v = semver.coerce("3.1.4-rc") # → Version("3.1.4-rc")
```

### semver.valid(version) → bool

Returns `True` if the string is a valid semver version (leading `v` is stripped before parsing).

### semver.coerce(version) → Version | None

Best-effort parsing of loose version strings. Missing minor/patch are filled with `0`. Returns `None` if the string cannot be parsed at all.

## Comparing versions

```python
import bunpy.semver as semver

a = semver.parse("1.2.3")
b = semver.parse("1.2.4")

a < b    # True
a > b    # False
a == b   # False
a <= b   # True

# Compare via string directly
semver.gt("2.0.0", "1.9.9")   # True
semver.lt("1.0.0", "1.0.1")   # True
semver.eq("1.2.3", "1.2.3")   # True
semver.gte("1.2.3", "1.2.3")  # True
semver.lte("1.2.2", "1.2.3")  # True
```

### Comparison functions

| Function | Description |
|----------|-------------|
| `semver.gt(a, b)` | `a > b` |
| `semver.lt(a, b)` | `a < b` |
| `semver.eq(a, b)` | `a == b` (build metadata ignored) |
| `semver.gte(a, b)` | `a >= b` |
| `semver.lte(a, b)` | `a <= b` |
| `semver.compare(a, b)` | `-1`, `0`, or `1` |
| `semver.diff(a, b)` | `"major"`, `"minor"`, `"patch"`, `"prerelease"`, or `None` |

```python
semver.diff("1.0.0", "2.0.0")   # "major"
semver.diff("1.0.0", "1.1.0")   # "minor"
semver.diff("1.0.0", "1.0.1")   # "patch"
semver.diff("1.0.0", "1.0.0")   # None
```

## Range checking

Ranges use the same syntax as npm/pip-style version ranges:

```python
import bunpy.semver as semver

semver.satisfies("1.2.3", ">=1.0.0 <2.0.0")   # True
semver.satisfies("2.0.0", ">=1.0.0 <2.0.0")   # False
semver.satisfies("1.2.3", "^1.0.0")            # True  - compatible with 1.x.x
semver.satisfies("2.0.0", "^1.0.0")            # False
semver.satisfies("1.2.3", "~1.2.0")            # True  - compatible with 1.2.x
semver.satisfies("1.3.0", "~1.2.0")            # False
semver.satisfies("1.2.3", "1.x")               # True
semver.satisfies("1.2.3", "*")                 # True
```

### Range operators

| Operator | Meaning |
|----------|---------|
| `^1.2.3` | `>=1.2.3 <2.0.0` - compatible changes |
| `~1.2.3` | `>=1.2.3 <1.3.0` - patch-level changes |
| `>=1.0.0 <2.0.0` | Intersection of two bounds |
| `1.x` | Any 1.y.z version |
| `1.2.x` | Any 1.2.z version |
| `*` | Any version |
| `1.2.3 \|\| 2.0.0` | Union |

### semver.satisfies(version, range) → bool

### semver.maxSatisfying(versions, range) → str | None

Returns the highest version in `versions` that satisfies `range`, or `None`.

```python
candidates = ["1.0.0", "1.2.3", "1.9.0", "2.0.0", "2.1.0"]
best = semver.maxSatisfying(candidates, "^1.0.0")
# "1.9.0"
```

### semver.minSatisfying(versions, range) → str | None

Returns the lowest satisfying version.

## Incrementing versions

```python
import bunpy.semver as semver

v = semver.parse("1.2.3")

semver.inc(v, "major")   # "2.0.0"
semver.inc(v, "minor")   # "1.3.0"
semver.inc(v, "patch")   # "1.2.4"

# Pre-release increment
semver.inc("1.2.3", "premajor", identifier="rc")  # "2.0.0-rc.0"
semver.inc("1.2.3", "preminor", identifier="beta") # "1.3.0-beta.0"
semver.inc("1.2.3", "prepatch")                    # "1.2.4-0"
semver.inc("1.2.3-rc.0", "prerelease")             # "1.2.3-rc.1"
```

### semver.inc(version, release, identifier=None) → str

| Release type | Example input | Output |
|-------------|--------------|--------|
| `"major"` | `1.2.3` | `2.0.0` |
| `"minor"` | `1.2.3` | `1.3.0` |
| `"patch"` | `1.2.3` | `1.2.4` |
| `"premajor"` | `1.2.3` | `2.0.0-0` |
| `"preminor"` | `1.2.3` | `1.3.0-0` |
| `"prepatch"` | `1.2.3` | `1.2.4-0` |
| `"prerelease"` | `1.2.3-rc.0` | `1.2.3-rc.1` |

## Sorting

```python
import bunpy.semver as semver

versions = ["1.0.0", "2.0.0", "1.5.0", "1.0.0-rc.1", "1.0.0-alpha"]
sorted_asc  = semver.sort(versions)
sorted_desc = semver.rsort(versions)

print(sorted_asc)
# ["1.0.0-alpha", "1.0.0-rc.1", "1.0.0", "1.5.0", "2.0.0"]
```

## Version gate checking

```python
import bunpy.semver as semver
import sys

REQUIRED_PYTHON = ">=3.12.0"

current = ".".join(str(x) for x in sys.version_info[:3])
if not semver.satisfies(current, REQUIRED_PYTHON):
    raise SystemExit(
        f"Python {REQUIRED_PYTHON} required, found {current}"
    )
```

## Changelog generation

```python
import bunpy.semver as semver
from bunpy.shell import sh

def changelog_since(tag: str) -> list[str]:
    result = sh(f"git log {tag}..HEAD --oneline", capture=True)
    return result.stdout.strip().splitlines()

def next_version(current: str, commits: list[str]) -> str:
    has_breaking = any("BREAKING" in c for c in commits)
    has_feat     = any(c.split(" ", 1)[1].startswith("feat") for c in commits if " " in c)

    if has_breaking:
        return semver.inc(current, "major")
    if has_feat:
        return semver.inc(current, "minor")
    return semver.inc(current, "patch")

current = "1.3.2"
commits = changelog_since(f"v{current}")
version = next_version(current, commits)

print(f"Next version: {version}")
print(f"Changes ({len(commits)}):")
for c in commits:
    print(f"  {c}")
```

## Dependency constraint logic

```python
import bunpy.semver as semver

def resolve(requested: str, available: list[str]) -> str | None:
    """Pick the best available version satisfying a range."""
    satisfying = [v for v in available if semver.satisfies(v, requested)]
    if not satisfying:
        return None
    return semver.rsort(satisfying)[0]   # highest satisfying

available = ["1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"]

print(resolve("^1.0.0", available))   # "1.2.0"
print(resolve("^2.0.0", available))   # "2.1.0"
print(resolve("~1.1.0", available))   # "1.1.0"
print(resolve(">=3.0.0", available))  # None
```

## Reference

| Function | Description |
|----------|-------------|
| `semver.parse(v)` | Parse string to `Version` object |
| `semver.valid(v)` | Check if string is valid semver |
| `semver.coerce(v)` | Loose parse, fill missing components with 0 |
| `semver.compare(a, b)` | `-1`, `0`, or `1` |
| `semver.diff(a, b)` | Component that changed |
| `semver.gt/lt/eq/gte/lte(a, b)` | Boolean comparisons |
| `semver.satisfies(v, range)` | Range check |
| `semver.maxSatisfying(vs, range)` | Highest satisfying version |
| `semver.minSatisfying(vs, range)` | Lowest satisfying version |
| `semver.inc(v, release)` | Increment to next version |
| `semver.sort(vs)` | Sort ascending |
| `semver.rsort(vs)` | Sort descending |
