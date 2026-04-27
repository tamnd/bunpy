// Package markdown renders Markdown text as ANSI-formatted terminal output.
package markdown

import (
	"regexp"
	"strings"
)

// ANSI codes.
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dim       = "\033[2m"
	underline = "\033[4m"
	cyan      = "\033[36m"
	grey      = "\033[90m"
)

var (
	reBold   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic = regexp.MustCompile(`(?:\*|_)(.+?)(?:\*|_)`)
	reCode   = regexp.MustCompile("`(.+?)`")
)

// Render converts Markdown src to ANSI-formatted terminal text.
func Render(src string) string {
	var out strings.Builder
	lines := strings.Split(src, "\n")
	inFence := false

	for _, line := range lines {
		// Code fence toggle.
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			if inFence {
				out.WriteString(dim)
			} else {
				out.WriteString(reset)
			}
			out.WriteByte('\n')
			continue
		}
		if inFence {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}

		// Headings.
		if strings.HasPrefix(line, "### ") {
			out.WriteString(bold + line[4:] + reset + "\n")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			out.WriteString(bold + line[3:] + reset + "\n\n")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			out.WriteString(bold + underline + line[2:] + reset + "\n\n")
			continue
		}

		// Blockquote.
		if strings.HasPrefix(line, "> ") {
			out.WriteString(grey + "│ " + inlineFormat(line[2:]) + reset + "\n")
			continue
		}

		// Bullet list.
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			out.WriteString("• " + inlineFormat(line[2:]) + "\n")
			continue
		}

		// Numbered list.
		if m := regexp.MustCompile(`^(\d+)\. (.+)`).FindStringSubmatch(line); m != nil {
			out.WriteString(m[1] + ". " + inlineFormat(m[2]) + "\n")
			continue
		}

		// Plain line (including blank).
		out.WriteString(inlineFormat(line) + "\n")
	}

	return out.String()
}

// inlineFormat applies bold, italic, and code spans.
func inlineFormat(line string) string {
	line = reBold.ReplaceAllString(line, bold+"$1"+reset)
	line = reItalic.ReplaceAllString(line, dim+"$1"+reset)
	line = reCode.ReplaceAllString(line, cyan+"$1"+reset)
	return line
}
