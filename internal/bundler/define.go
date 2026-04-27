package bundler

import (
	"fmt"
	"regexp"
)

// applyDefines replaces whole-word occurrences of each key with its value.
func applyDefines(src string, defines map[string]string) string {
	for k, v := range defines {
		re := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(k)))
		src = re.ReplaceAllString(src, v)
	}
	return src
}
