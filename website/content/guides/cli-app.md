---
title: Build a CLI app
description: Parse arguments, read stdin, and return exit codes.
---

## Basic CLI with argparse

```python
import argparse
import sys

parser = argparse.ArgumentParser(description="My CLI tool")
parser.add_argument("name", help="Who to greet")
parser.add_argument("--count", type=int, default=1, help="How many times")
parser.add_argument("--upper", action="store_true", help="UPPERCASE output")

args = parser.parse_args()

for _ in range(args.count):
    msg = f"Hello, {args.name}!"
    print(msg.upper() if args.upper else msg)
```

```bash
bunpy greet.py Alice
# Hello, Alice!

bunpy greet.py Alice --count 3 --upper
# HELLO, ALICE!
# HELLO, ALICE!
# HELLO, ALICE!
```

## Reading stdin

```python
import sys

for line in sys.stdin:
    print(line.strip().upper())
```

```bash
echo -e "hello\nworld" | bunpy upper.py
# HELLO
# WORLD
```

## Exit codes

```python
import sys

def main():
    if len(sys.argv) < 2:
        print("Usage: myscript.py <file>", file=sys.stderr)
        sys.exit(1)
    print(f"Processing {sys.argv[1]}")

main()
```

## Bundle as a single file

```bash
bunpy build greet.py -o greet.pyz
./greet.pyz Alice
```

Or compile to a native binary:

```bash
bunpy build --compile greet.py -o greet
./greet Alice
```
