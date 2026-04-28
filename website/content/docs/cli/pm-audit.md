---
title: bunpy pm audit
description: Check packages in uv.lock against the OSV vulnerability database.
weight: 23
---

```bash
bunpy pm audit [flags]
```

## Description

`bunpy pm audit` checks every package pinned in `uv.lock` against the [OSV (Open Source Vulnerabilities)](https://osv.dev/) database — the same source used by GitHub Dependabot and `pip audit`. It sends the list of `(name, version)` pairs to the OSV batch API and reports any matching advisories.

No packages are modified. `pm audit` is a read-only check.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--lockfile <path>` | `uv.lock` | Lock file to audit |
| `--json` | off | Output results as JSON |
| `--ignore <id,...>` | | Comma-separated OSV IDs to suppress |
| `--severity <level>` | all | Minimum severity to report: `low`, `medium`, `high`, `critical` |
| `--help`, `-h` | | Print help |

## Output

When vulnerabilities are found:

```bash
bunpy pm audit
```

```
Auditing 47 packages against OSV database...

CRITICAL  requests 2.19.1
  CVE-2023-32681  Unintended leak of Proxy-Authorization header
  Fix: upgrade to >=2.31.0
  https://osv.dev/vulnerability/GHSA-j8r2-6x86-q33q

HIGH      cryptography 41.0.0
  CVE-2023-49083  NULL pointer dereference in PKCS12 parsing
  Fix: upgrade to >=41.0.6
  https://osv.dev/vulnerability/GHSA-jm77-qphf-c4w8

HIGH      cryptography 41.0.0
  CVE-2023-23931  Blowfish cipher allows weak key sizes
  Fix: upgrade to >=39.0.1
  https://osv.dev/vulnerability/GHSA-w7pp-m8wf-vj6r

3 vulnerabilities found in 2 packages.
Run `bunpy update requests cryptography` to apply fixes.
```

When the project is clean:

```
Auditing 47 packages against OSV database...
No vulnerabilities found.
```

### Output columns

| Column | Description |
|--------|-------------|
| Severity | `CRITICAL`, `HIGH`, `MEDIUM`, or `LOW` from OSV |
| Package | Name and installed version |
| CVE | CVE ID or GHSA ID for the advisory |
| Description | One-line summary |
| Fix | Minimum safe version, if the advisory provides one |
| URL | Link to the full OSV advisory |

## JSON output

```bash
bunpy pm audit --json
```

```json
{
  "audited": 47,
  "vulnerabilities": [
    {
      "package": "requests",
      "version": "2.19.1",
      "id": "GHSA-j8r2-6x86-q33q",
      "aliases": ["CVE-2023-32681"],
      "severity": "CRITICAL",
      "summary": "Unintended leak of Proxy-Authorization header",
      "fix": ">=2.31.0",
      "url": "https://osv.dev/vulnerability/GHSA-j8r2-6x86-q33q"
    },
    {
      "package": "cryptography",
      "version": "41.0.0",
      "id": "GHSA-jm77-qphf-c4w8",
      "aliases": ["CVE-2023-49083"],
      "severity": "HIGH",
      "summary": "NULL pointer dereference in PKCS12 parsing",
      "fix": ">=41.0.6",
      "url": "https://osv.dev/vulnerability/GHSA-jm77-qphf-c4w8"
    }
  ]
}
```

`bunpy pm audit` exits with code 1 when any vulnerabilities are found, 0 when clean. Scripts and CI pipelines can branch on the exit code without parsing output.

## CI integration

Add an audit step to every CI run so vulnerabilities are caught before code ships:

```yaml
# .github/workflows/ci.yml
- name: Audit dependencies
  run: bunpy pm audit
```

For teams that want to audit without blocking merges (visibility-first), suppress the exit code:

```bash
bunpy pm audit || echo "::warning::Vulnerabilities found — review pm audit output"
```

### Filtering by severity

Only block on high and critical findings:

```bash
bunpy pm audit --severity high
```

This lets low and medium advisories surface in logs without failing the build.

### Ignoring known false positives

If an advisory is a known false positive or you have a compensating control, suppress it by OSV ID:

```bash
bunpy pm audit --ignore GHSA-j8r2-6x86-q33q,GHSA-jm77-qphf-c4w8
```

Document suppressed IDs in a comment near the command so future maintainers understand why they were ignored.

## What to do when vulnerabilities are found

1. **Check the fix version.** The output shows the minimum safe version. Run `bunpy update <package>` to upgrade to the latest version that satisfies your constraints.

2. **Verify your constraints allow the fix.** If `pyproject.toml` pins `cryptography = ">=41.0.0,<41.0.5"` and the fix is `>=41.0.6`, widen the constraint first.

3. **Re-run audit.** After updating, run `bunpy pm audit` again to confirm the advisory is resolved.

4. **If no fix version exists**, the advisory may be informational or the project may be abandoned. Evaluate whether the code path that triggers the vulnerability is reachable in your application, and consider switching to an alternative package.

## Data source

All advisory data comes from `https://api.osv.dev/v1/querybatch`. The OSV dataset aggregates advisories from NVD, GitHub Advisory Database, PyPA, and others. No data is sent beyond package names and versions — no source code, no file paths, no credentials.
