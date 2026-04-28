---
title: Installation
description: Install bunpy on macOS, Linux, or Windows. One command to get a Python runtime, package manager, and bundler.
weight: 1
---

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
```

That's the install. One command, no prerequisites. bunpy is a single static binary written in Go -- it ships its own Python 3.14 runtime (goipy), its own package manager, and its own bundler. There is nothing else to install.

## macOS and Linux

Run the install script:

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
```

The script:
1. Detects your OS and CPU architecture.
2. Downloads the correct binary from the [releases page](https://github.com/tamnd/bunpy/releases/latest).
3. Places it at `~/.bunpy/bin/bunpy`.
4. Prints instructions to update your `PATH`.

Add bunpy to your `PATH` permanently by appending this line to `~/.bashrc`, `~/.zshrc`, or whatever profile your shell loads:

```bash
export PATH="$HOME/.bunpy/bin:$PATH"
```

Then reload your shell:

```bash
source ~/.zshrc   # or ~/.bashrc
```

### Verify

```bash
bunpy --version
# bunpy 0.10.29 (linux/amd64)
```

### Supported platforms

| Platform | Architecture |
|----------|-------------|
| macOS | x86_64, arm64 (Apple Silicon) |
| Linux | x86_64, arm64 |
| Windows | x86_64 |

## Homebrew (macOS and Linux)

If you prefer Homebrew:

```bash
brew tap tamnd/bunpy
brew install bunpy
```

Homebrew manages upgrades through the normal `brew upgrade` flow.

## Windows

Download the latest `.zip` archive for Windows from the [releases page](https://github.com/tamnd/bunpy/releases/latest). Extract it and add the folder containing `bunpy.exe` to your system `PATH`.

Using PowerShell (run as Administrator):

```powershell
$dest = "$env:USERPROFILE\.bunpy\bin"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
# Copy bunpy.exe to $dest, then:
[Environment]::SetEnvironmentVariable("PATH", "$dest;" + $env:PATH, "User")
```

Restart your terminal and verify:

```powershell
bunpy --version
```

## Upgrading

```bash
bunpy upgrade
```

This downloads and replaces the current binary in place. Re-running the install script has the same effect.

To upgrade to a specific version:

```bash
bunpy upgrade --version 0.10.29
```

## Pinning a version

You can pin a specific release by passing `BUNPY_VERSION` to the install script:

```bash
BUNPY_VERSION=0.10.29 curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
```

This is useful in CI environments where you want reproducible builds regardless of what the latest release is.

## Docker

The official image is available on Docker Hub:

```dockerfile
FROM tamnd/bunpy:latest

WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen

COPY . .
CMD ["bunpy", "run", "src/main.py"]
```

For a specific version:

```dockerfile
FROM tamnd/bunpy:0.10.29
```

The image is Alpine-based and weighs around 80 MB compressed.

## CI environments

### GitHub Actions

```yaml
- name: Install bunpy
  run: curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
- name: Add bunpy to PATH
  run: echo "$HOME/.bunpy/bin" >> $GITHUB_PATH
```

### GitLab CI

```yaml
before_script:
  - curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
  - export PATH="$HOME/.bunpy/bin:$PATH"
```

## Uninstalling

```bash
rm -rf ~/.bunpy
```

Then remove the `export PATH` line from your shell profile. If you installed via Homebrew, use `brew uninstall bunpy` instead.

## What is installed

| Path | Contents |
|------|----------|
| `~/.bunpy/bin/bunpy` | The bunpy binary |
| `~/.cache/bunpy/wheels/` | Downloaded wheel cache (populated on first `bunpy install`) |

The binary is self-contained. All Python stdlib modules are embedded inside it -- no system Python is required, and bunpy does not use your system Python even if one is present.
