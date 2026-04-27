// Package checker runs fast static lint rules on Python source.
package checker

import (
	"fmt"
	"regexp"
	"strings"
)

// Issue is a single lint finding.
type Issue struct {
	File    string
	Line    int
	Code    string
	Message string
}

func (i Issue) String() string {
	return fmt.Sprintf("%s:%d: %s %s", i.File, i.Line, i.Code, i.Message)
}

var (
	reBareExcept    = regexp.MustCompile(`^\s*except\s*:`)
	reNoneEq        = regexp.MustCompile(`==\s*None|None\s*==`)
	reBoolEq        = regexp.MustCompile(`==\s*(True|False)|(True|False)\s*==`)
	rePrint2        = regexp.MustCompile(`^\s*print\s+[^(]`)
	reImport        = regexp.MustCompile(`^\s*(?:import|from)\s+(\w[\w.]*)`)
	reIdentifier    = regexp.MustCompile(`\b([A-Za-z_]\w*)\b`)
)

// Check runs all lint rules against src and returns any issues found.
func Check(filename, src string) []Issue {
	var issues []Issue
	lines := strings.Split(src, "\n")

	// Track imported names for E005.
	imported := map[string]int{} // name → line number

	for i, line := range lines {
		lineNum := i + 1

		// W001: trailing whitespace
		if strings.TrimRight(line, " \t") != line && strings.TrimRight(line, "\r\n") != "" {
			issues = append(issues, Issue{filename, lineNum, "W001", "trailing whitespace"})
		}

		// W002: line too long (>120 chars)
		if len(line) > 120 {
			issues = append(issues, Issue{filename, lineNum, "W002", fmt.Sprintf("line too long (%d chars)", len(line))})
		}

		// E001: bare except
		if reBareExcept.MatchString(line) {
			issues = append(issues, Issue{filename, lineNum, "E001", "bare except: catches all exceptions"})
		}

		// E002: Python 2 print statement
		if rePrint2.MatchString(line) {
			issues = append(issues, Issue{filename, lineNum, "E002", "print used as statement (missing parentheses?)"})
		}

		// E003: == None
		if reNoneEq.MatchString(line) {
			issues = append(issues, Issue{filename, lineNum, "E003", "use 'is None' instead of '== None'"})
		}

		// E004: == True / == False
		if reBoolEq.MatchString(line) {
			issues = append(issues, Issue{filename, lineNum, "E004", "use 'is True'/'is False' instead of '== True'/'== False'"})
		}

		// Track imports for E005.
		if m := reImport.FindStringSubmatch(line); m != nil {
			// Only track the top-level module name.
			name := strings.SplitN(m[1], ".", 2)[0]
			if _, seen := imported[name]; !seen {
				imported[name] = lineNum
			}
		}
	}

	// E005: unused import — name never appears outside import lines.
	for name, importLine := range imported {
		used := false
		for i, line := range lines {
			lineNum := i + 1
			if lineNum == importLine {
				continue
			}
			for _, m := range reIdentifier.FindAllString(line, -1) {
				if m == name {
					used = true
					break
				}
			}
			if used {
				break
			}
		}
		if !used {
			issues = append(issues, Issue{filename, importLine, "E005", fmt.Sprintf("imported name %q is never used", name)})
		}
	}

	return issues
}
