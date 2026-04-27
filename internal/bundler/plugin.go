package bundler

import (
	"fmt"
	"io"
	"os"
)

// ApplyPlugins runs each plugin file against the bundle's source files.
// Plugins are Python files that define transform(source, filename) -> str.
// Since gocopy does not yet support function definitions, plugins fall back
// to no-op with a warning printed to warn.
func ApplyPlugins(b *Bundle, plugins []string, warn io.Writer) {
	for _, pluginPath := range plugins {
		if _, err := os.Stat(pluginPath); err != nil {
			fmt.Fprintf(warn, "warning: plugin %q not found, skipping\n", pluginPath)
			continue
		}
		// Plugin execution requires gocopy function-definition support (v0.1.x).
		// Until then, emit a warning and skip.
		fmt.Fprintf(warn, "warning: plugin %q requires gocopy function-definition support; skipping\n", pluginPath)
	}
}
