// Package repl implements `bunpy repl`, the interactive line-driver
// shell on top of the goipy bytecode VM.
//
// v0.0.8 is stateless: every flushed chunk is handed to runtime.Run
// as a fresh module. Persistent globals across chunks waits for
// gocopy to grow expression and call compilation. The CLI surface
// (prompt, multi-line buffers, history, meta commands) is stable so
// the language story can grow under it without breaking callers.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/tamnd/bunpy/v1/runtime"
)

// Options configures a Loop. All fields are optional; zero values
// give the default interactive behaviour.
type Options struct {
	// Quiet suppresses the startup banner and `>>>` / `... `
	// prompts. Useful for piped-stdin smokes and for fixtures.
	Quiet bool

	// HistoryPath, when non-empty, is loaded on start and appended
	// on every successful flush.
	HistoryPath string

	// HistorySize caps the in-memory and on-disk history length.
	// Zero disables history entirely. Negative values are treated
	// as zero.
	HistorySize int

	// Filename overrides the synthetic filename passed into
	// runtime.Run. Defaults to "<repl>".
	Filename string

	// runner is the eval hook. Tests inject a stub; the default
	// uses runtime.Run.
	runner func(filename string, source []byte, stdout, stderr io.Writer) (int, error)
}

// Loop reads from stdin, evaluates chunks, and writes prompts and
// banner to stdout, errors to stderr. Returns 0 on a clean :quit or
// EOF, non-zero on an internal error (not on user-side compile
// errors, which are reported and recovered from).
func Loop(stdin io.Reader, stdout, stderr io.Writer, opts Options) int {
	if opts.runner == nil {
		opts.runner = func(filename string, source []byte, out, errw io.Writer) (int, error) {
			return runtime.Run(filename, source, nil, out, errw)
		}
	}
	if opts.Filename == "" {
		opts.Filename = "<repl>"
	}

	hist := loadHistory(opts.HistoryPath, opts.HistorySize)

	if !opts.Quiet {
		b := runtime.Build()
		fmt.Fprintf(stdout, "bunpy %s repl. Type :help for commands, :quit to exit.\n", b.Version)
	}

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	var buf []string
	prompt := func() {
		if opts.Quiet {
			return
		}
		if len(buf) == 0 {
			fmt.Fprint(stdout, ">>> ")
		} else {
			fmt.Fprint(stdout, "... ")
		}
	}

	prompt()
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, ":") {
			cont := dispatchMeta(line, &buf, hist, stdout, stderr)
			if !cont {
				return 0
			}
			prompt()
			continue
		}

		if line == "" {
			if len(buf) == 0 {
				prompt()
				continue
			}
			source := strings.Join(buf, "\n") + "\n"
			_, err := opts.runner(opts.Filename, []byte(source), stdout, stderr)
			if err != nil {
				fmt.Fprintln(stderr, "bunpy repl:", err)
			} else {
				hist.append(strings.Join(buf, "\n"))
			}
			buf = buf[:0]
			prompt()
			continue
		}

		buf = append(buf, line)
		prompt()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(stderr, "bunpy repl:", err)
		return 1
	}

	if len(buf) > 0 {
		source := strings.Join(buf, "\n") + "\n"
		if _, err := opts.runner(opts.Filename, []byte(source), stdout, stderr); err != nil {
			fmt.Fprintln(stderr, "bunpy repl:", err)
		} else {
			hist.append(strings.Join(buf, "\n"))
		}
	}
	return 0
}

// dispatchMeta handles a `:command` line. Returns false to terminate
// the loop, true to continue.
func dispatchMeta(line string, buf *[]string, hist *history, stdout, stderr io.Writer) bool {
	fields := strings.Fields(line)
	cmd := fields[0]
	args := fields[1:]
	switch cmd {
	case ":quit", ":exit":
		return false
	case ":help":
		fmt.Fprintln(stdout, "REPL commands:")
		fmt.Fprintln(stdout, "  :help            this message")
		fmt.Fprintln(stdout, "  :quit, :exit     leave the REPL")
		fmt.Fprintln(stdout, "  :history [N]     print the last N entries (default all)")
		fmt.Fprintln(stdout, "  :clear           drop the in-flight buffer")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Each blank line flushes the buffer through bunpy run.")
		fmt.Fprintln(stdout, "v0.0.8 is stateless: chunks do not share globals.")
		return true
	case ":history":
		n := -1
		if len(args) > 0 {
			if v, ok := parseInt(args[0]); ok {
				n = v
			}
		}
		hist.print(stdout, n)
		return true
	case ":clear":
		*buf = (*buf)[:0]
		return true
	default:
		fmt.Fprintf(stderr, "unknown REPL command %q (try :help)\n", cmd)
		return true
	}
}

func parseInt(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
