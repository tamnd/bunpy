package bundler

import (
	"strings"
)

// minifySource strips blank lines and comments from Python source.
// It uses a simple state machine to avoid stripping # inside string literals.
func minifySource(src string) string {
	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Strip inline comment (not inside a string).
		line = stripInlineComment(line)
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// stripInlineComment removes a trailing # comment from a line of Python,
// avoiding false positives inside string literals.
func stripInlineComment(line string) string {
	var inSingle, inDouble bool
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '#' && !inSingle && !inDouble:
			return strings.TrimRight(line[:i], " \t")
		}
	}
	return line
}
