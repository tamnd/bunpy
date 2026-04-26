# Changelog

All notable changes to bunpy are recorded here. The format follows
[Keep a Changelog 1.1](https://keepachangelog.com/en/1.1.0/). Once
bunpy reaches 1.0 the project will follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html); until
then, expect minor version bumps to sometimes include breaking
changes.

## [Unreleased]

## [0.0.1] - 2026-04-26

Bootstrap. The repo is the skeleton: README, LICENSE, the CLI
entry point with `bunpy --version` and `bunpy --help`, the CI
workflow set (lint, test, cross-platform build, tag-driven
release), the changelog tooling, and the docs landing pages.

There is no runtime yet. `bunpy <file.py>` is reserved for
v0.0.2, which wires goipy in. The ladder is in
[`docs/ROADMAP.md`](docs/ROADMAP.md).

### Added

- `cmd/bunpy/main.go` with `bunpy version`, `--version`,
  `bunpy help`, `--help`, and a subcommand router shaped for the
  rungs ahead.
- `LICENSE` (MIT, dated 2026-04-26).
- `README.md` with the Bun-to-bunpy CLI map and the Python API
  shape.
- `.github/workflows/ci.yml` runs `go vet`, `go build`, and
  `go test ./...` on linux, macOS, and Windows.
- `.github/workflows/build.yml` is a cross-compile sanity
  matrix (linux/darwin/windows over amd64/arm64).
- `.github/workflows/release.yml` does the tag-driven build,
  produces archives plus SHA-256 checksums, and creates a
  GitHub release whose body is `changelog/${tag}.md`.
- `scripts/build-changelog.sh` concatenates `changelog/v*.md`
  into `CHANGELOG.md`, gopapy-style.
- `scripts/feature-coverage.sh` plus `scripts/coverage.tsv`
  generate the bunpy-vs-Bun coverage table for
  `docs/COVERAGE.md`.
- `docs/` landing pages: `ARCHITECTURE.md`, `ROADMAP.md`,
  `CLI.md`, `API.md`, `COVERAGE.md`, `COMPATIBILITY.md`,
  `DEVIATIONS.md`.
- `.gitignore`, `.editorconfig`.

### Notes

bunpy targets Python 3.14 because gopapy, gocopy, and goipy do.

