package bundler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type sourceMapFile struct {
	Version int               `json:"version"`
	Sources []sourceMapEntry  `json:"sources"`
}

type sourceMapEntry struct {
	Bundled  string `json:"bundled"`
	Original string `json:"original"`
	Lines    int    `json:"lines"`
}

// WriteSourceMap writes a JSON source map alongside the .pyz.
func WriteSourceMap(sources []SourceEntry, outpath string) error {
	mapPath := outpath + ".map"
	sm := sourceMapFile{Version: 1}
	for _, s := range sources {
		sm.Sources = append(sm.Sources, sourceMapEntry{
			Bundled:  s.Bundled,
			Original: s.Original,
			Lines:    s.Lines,
		})
	}
	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return fmt.Errorf("sourcemap: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(mapPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(mapPath, data, 0o644)
}
