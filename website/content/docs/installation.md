---
title: Installation
description: Install bunpy on macOS, Linux, or Windows.
weight: 1
---

## macOS and Linux

Install bunpy with a single curl command:

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
```

This places the `bunpy` binary in `~/.bunpy/bin`. Add it to your PATH:

```bash
export PATH="$HOME/.bunpy/bin:$PATH"
```

Add that line to your `~/.bashrc` or `~/.zshrc` to make it permanent.

## Homebrew (macOS and Linux)

```bash
brew tap tamnd/bunpy
brew install bunpy
```

## Windows

Download the latest `.zip` for Windows from the [releases page](https://github.com/tamnd/bunpy/releases/latest), extract it, and add the folder to your `PATH`.

## Verify the installation

```bash
bunpy --version
```

You should see output like:

```
bunpy 0.9.0 (linux/amd64)
```

## Upgrading

```bash
bunpy upgrade
```

Or re-run the install script — it replaces the existing binary.

## Uninstalling

```bash
rm -rf ~/.bunpy
```

Remove the `PATH` entry from your shell profile file.
