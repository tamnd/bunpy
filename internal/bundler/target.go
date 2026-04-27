package bundler

import (
	"archive/zip"
	"fmt"
	"runtime"
)

var validTargets = map[string]bool{
	"linux-x64":    true,
	"linux-arm64":  true,
	"darwin-x64":   true,
	"darwin-arm64": true,
	"windows-x64":  true,
	"browser":      true,
}

// ValidateTarget returns an error if target is unknown.
func ValidateTarget(target string) error {
	if target == "" {
		return nil
	}
	if !validTargets[target] {
		return fmt.Errorf("bundler: unknown target %q; valid: linux-x64, linux-arm64, darwin-x64, darwin-arm64, windows-x64, browser", target)
	}
	return nil
}

// CurrentTarget returns the target string for the current host.
func CurrentTarget() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return os + "-" + arch
}

func writeTargetMetadata(zw *zip.Writer, target string) error {
	w, err := zw.Create("METADATA")
	if err != nil {
		return err
	}
	version := "0.6.0"
	fmt.Fprintf(w, "bunpy-target: %s\nbunpy-version: %s\n", target, version)

	if target == "browser" {
		wb, err := zw.Create("WASM_NOTE")
		if err != nil {
			return err
		}
		fmt.Fprintln(wb, "Browser/WASM target is planned for a future release.")
		fmt.Fprintln(wb, "This archive was built for documentation purposes only.")
	}
	return nil
}
