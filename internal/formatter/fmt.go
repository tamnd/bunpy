// Package formatter normalises Python source files.
package formatter

import (
	"bytes"
	"strings"
)

// Format normalises src:
//   - CRLF and CR line endings → LF
//   - trailing whitespace stripped from each line
//   - leading tabs replaced with 4 spaces
//   - exactly one trailing newline at end of file
func Format(src []byte) []byte {
	// Normalise line endings.
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	var out bytes.Buffer
	for _, line := range lines {
		line = expandLeadingTabs(line)
		line = strings.TrimRight(line, " \t")
		out.WriteString(line)
		out.WriteByte('\n')
	}

	// Collapse multiple trailing newlines to exactly one.
	result := bytes.TrimRight(out.Bytes(), "\n")
	result = append(result, '\n')
	return result
}

// Changed reports whether Format(src) != src.
func Changed(src []byte) bool {
	return !bytes.Equal(Format(src), src)
}

func expandLeadingTabs(line string) string {
	i := 0
	for i < len(line) && line[i] == '\t' {
		i++
	}
	if i == 0 {
		return line
	}
	return strings.Repeat("    ", i) + line[i:]
}
