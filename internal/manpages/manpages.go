// Package manpages embeds bunpy's roff manpages so they ride
// inside the binary and can be served via `bunpy man <cmd>` or
// installed with `bunpy man --install`.
package manpages

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

//go:embed man1/*.1
var fsys embed.FS

// Root is the directory inside the embed.FS that holds the pages.
const Root = "man1"

// FS returns the embedded filesystem rooted at Root.
func FS() fs.FS { return fsys }

// Page returns the roff bytes for the named subcommand: "" or
// "bunpy" returns bunpy.1; "bunpyx" returns bunpyx.1; otherwise
// bunpy-<name>.1.
func Page(name string) ([]byte, error) {
	if name == "" || name == "bunpy" {
		return fsys.ReadFile(Root + "/bunpy.1")
	}
	prefixed := Root + "/bunpy-" + name + ".1"
	if data, err := fsys.ReadFile(prefixed); err == nil {
		return data, nil
	}
	return fsys.ReadFile(Root + "/" + name + ".1")
}

// List returns the embedded manpage filenames, sorted.
func List() ([]string, error) {
	entries, err := fsys.ReadDir(Root)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".1") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}
