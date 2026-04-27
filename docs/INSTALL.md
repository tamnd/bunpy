# Installing bunpy

bunpy ships as a single static Go binary. Three install paths:
the one-liner script, the Homebrew tap, or the manual download.

## One-liner (linux, macOS)

```
curl -fsSL https://raw.githubusercontent.com/tamnd/bunpy/main/install.sh | bash
```

The script downloads the latest release archive matching your
os/arch, verifies its SHA-256 against the release's
`SHA256SUMS`, and installs the binary at
`$HOME/.bunpy/bin/bunpy`. Add that directory to `PATH`:

```
export PATH="$HOME/.bunpy/bin:$PATH"
```

Re-running the same one-liner upgrades in place. The previous
binary is preserved at `$HOME/.bunpy/bin/bunpy.prev` so a
rollback is one `mv` away.

### Env overrides

- `BUNPY_VERSION=v0.0.6` pin to a specific tag instead of latest.
- `BUNPY_INSTALL_DIR=/opt/bunpy` change install root.
- `BUNPY_OS=linux`, `BUNPY_ARCH=amd64` override autodetection.

## Homebrew (macOS, linux)

```
brew install tamnd/bunpy/bunpy
```

The formula lives at `tamnd/homebrew-bunpy`. The release workflow
pushes a fresh formula to that tap on every tag, so
`brew upgrade bunpy` always pulls the most recent release.

## Manual download

Releases page: <https://github.com/tamnd/bunpy/releases>. Each
tag ships six archives:

- `bunpy-vX.Y.Z-linux-amd64.tar.gz`
- `bunpy-vX.Y.Z-linux-arm64.tar.gz`
- `bunpy-vX.Y.Z-darwin-amd64.tar.gz`
- `bunpy-vX.Y.Z-darwin-arm64.tar.gz`
- `bunpy-vX.Y.Z-windows-amd64.zip`
- `bunpy-vX.Y.Z-windows-arm64.zip`

Plus a `SHA256SUMS` file listing the SHA-256 of each archive.
Verify before extracting:

```
shasum -a 256 -c SHA256SUMS
tar -xzf bunpy-vX.Y.Z-linux-amd64.tar.gz
sudo mv bunpy-vX.Y.Z-linux-amd64/bunpy /usr/local/bin/
```

Windows users: extract the zip and add the resulting directory
to `PATH`.

## Verifying

After install, `bunpy version` prints the version, the commit
the binary was built from, the build date, and the three pinned
sibling-toolchain commits (gopapy, gocopy, goipy). For scripts:

```
bunpy version --short    # just the version
bunpy version --json     # one-line JSON
```
