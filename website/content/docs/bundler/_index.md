---
title: Bundler
description: Bundle Python scripts to .pyz archives or native binaries.
weight: 6
---

`bunpy build` compiles a Python entry point and its imports into a portable
artifact. Two output modes:

| Mode | Output | Requires bunpy to run? |
|------|--------|------------------------|
| `.pyz` | ZIP archive with `__main__.py` | Yes |
| `--compile` | Native binary (embeds goipy VM) | No |

{{< cards >}}
  {{< card link="pyz" title=".pyz format" subtitle="Portable archive that runs with bunpy" >}}
  {{< card link="minify" title="Minify" subtitle="Strip whitespace and comments at bundle time" >}}
  {{< card link="sourcemaps" title="Source maps" subtitle="Debug bundled code with .pyz.map files" >}}
  {{< card link="define" title="Compile-time defines" subtitle="Replace constants at build time" >}}
  {{< card link="targets" title="Build targets" subtitle="Cross-compile for linux, darwin, windows" >}}
  {{< card link="watch" title="Watch mode" subtitle="Rebuild automatically on file changes" >}}
  {{< card link="compile" title="Compile to binary" subtitle="Ship a self-contained native executable" >}}
{{< /cards >}}
