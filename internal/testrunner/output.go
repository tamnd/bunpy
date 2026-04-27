package testrunner

import (
	"fmt"
	"io"
	"strings"
)

// Printer writes test results to an io.Writer.
type Printer struct {
	W       io.Writer
	Verbose bool
	NoColor bool
}

const (
	green  = "\x1b[32m"
	red    = "\x1b[31m"
	yellow = "\x1b[33m"
	gray   = "\x1b[90m"
	bold   = "\x1b[1m"
	reset  = "\x1b[0m"
)

func (p *Printer) color(code, s string) string {
	if p.NoColor {
		return s
	}
	return code + s + reset
}

// PrintFileResult writes the result of one file to the writer.
func (p *Printer) PrintFileResult(fr FileResult) {
	if fr.CompileError != "" {
		fmt.Fprintf(p.W, "%s %s\n", p.color(red, "ERROR"), fr.File)
		for _, line := range strings.Split(fr.CompileError, "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintf(p.W, "    %s\n", p.color(gray, line))
			}
		}
		return
	}

	pass := fr.Pass()
	icon := p.color(green, "PASS")
	if !pass {
		icon = p.color(red, "FAIL")
	}
	fmt.Fprintf(p.W, "%s %s %s\n", icon, fr.File,
		p.color(gray, fmt.Sprintf("(%s)", fr.Duration.Round(1000000))))

	for _, r := range fr.Results {
		p.printResult(r)
	}
}

func (p *Printer) printResult(r TestResult) {
	switch r.Status {
	case StatusPass:
		if p.Verbose {
			fmt.Fprintf(p.W, "  %s %s %s\n",
				p.color(green, "✓"),
				r.Name,
				p.color(gray, fmt.Sprintf("(%s)", r.Duration.Round(1000000))))
		}
	case StatusFail:
		fmt.Fprintf(p.W, "  %s %s\n", p.color(red, "✗"), p.color(bold, r.Name))
		if r.Message != "" {
			for _, line := range strings.Split(r.Message, "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Fprintf(p.W, "    %s\n", p.color(red, line))
				}
			}
		}
	case StatusSkip:
		if p.Verbose {
			fmt.Fprintf(p.W, "  %s %s\n", p.color(yellow, "○"), r.Name)
		}
	case StatusError:
		fmt.Fprintf(p.W, "  %s %s\n", p.color(red, "!"), p.color(bold, r.Name))
		if r.Message != "" {
			for _, line := range strings.Split(r.Message, "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Fprintf(p.W, "    %s\n", p.color(red, line))
				}
			}
		}
	}
}

// PrintSummary writes the aggregate run summary.
func (p *Printer) PrintSummary(s Summary) {
	fmt.Fprintln(p.W)
	parts := []string{
		fmt.Sprintf("%d passed", s.Passed),
	}
	if s.Failed > 0 {
		parts = append(parts, p.color(red, fmt.Sprintf("%d failed", s.Failed)))
	}
	if s.Errors > 0 {
		parts = append(parts, p.color(red, fmt.Sprintf("%d errored", s.Errors)))
	}
	if s.Skipped > 0 {
		parts = append(parts, p.color(yellow, fmt.Sprintf("%d skipped", s.Skipped)))
	}
	total := fmt.Sprintf("%d tests", s.Total)
	if s.AllPassed() {
		fmt.Fprintf(p.W, "%s  %s  %s\n",
			p.color(green, "Tests:"),
			strings.Join(parts, ", "),
			p.color(gray, total))
	} else {
		fmt.Fprintf(p.W, "%s  %s  %s\n",
			p.color(red, "Tests:"),
			strings.Join(parts, ", "),
			p.color(gray, total))
	}
	fmt.Fprintf(p.W, "%s  %s\n",
		p.color(gray, "Duration:"),
		s.Duration.Round(1000000))
}
