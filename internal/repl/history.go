package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// history is the in-memory ring backing :history and the persistent
// file write. cap=0 disables both; the in-memory list still tracks
// entries for :history so the same session can read what it typed.
type history struct {
	path    string
	cap     int
	entries []string
}

func loadHistory(path string, cap int) *history {
	h := &history{path: path, cap: cap}
	if cap <= 0 || path == "" {
		return h
	}
	f, err := os.Open(path)
	if err != nil {
		return h
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		entry := scanner.Text()
		// Multi-line entries were saved with literal "\n" sequences
		// to keep one entry per line on disk.
		entry = strings.ReplaceAll(entry, `\n`, "\n")
		h.entries = append(h.entries, entry)
	}
	if len(h.entries) > cap {
		h.entries = h.entries[len(h.entries)-cap:]
	}
	return h
}

func (h *history) append(entry string) {
	if h.cap <= 0 {
		return
	}
	h.entries = append(h.entries, entry)
	if len(h.entries) > h.cap {
		h.entries = h.entries[len(h.entries)-h.cap:]
	}
	if h.path == "" {
		return
	}
	tmp, err := os.CreateTemp(dirOf(h.path), ".bunpy_history.*")
	if err != nil {
		return
	}
	w := bufio.NewWriter(tmp)
	for _, e := range h.entries {
		fmt.Fprintln(w, strings.ReplaceAll(e, "\n", `\n`))
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return
	}
	_ = os.Rename(tmp.Name(), h.path)
}

func (h *history) print(out io.Writer, n int) {
	if len(h.entries) == 0 {
		return
	}
	start := 0
	if n > 0 && n < len(h.entries) {
		start = len(h.entries) - n
	}
	for i := start; i < len(h.entries); i++ {
		fmt.Fprintf(out, "%d  %s\n", i+1, h.entries[i])
	}
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return "."
}
