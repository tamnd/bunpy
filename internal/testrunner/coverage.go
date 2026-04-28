package testrunner

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CoverageInfo holds simple coverage data for one file.
type CoverageInfo struct {
	File       string
	TotalLines int
	ExecLines  int // lines that were executed (estimated from test count)
}

// Coverage returns coverage percentage (0-100).
func (c CoverageInfo) Coverage() float64 {
	if c.TotalLines == 0 {
		return 100.0
	}
	return float64(c.ExecLines) / float64(c.TotalLines) * 100
}

// WriteCoverage writes a coverage report to coverDir and prints a summary.
// When coll is non-nil it uses real line-trace hit data; otherwise it falls
// back to the static estimate (files that have a matching test file = 70%).
func WriteCoverage(testFiles []string, coverDir string, stdout io.Writer, coll *CoverageCollector) error {
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		return fmt.Errorf("coverage: mkdir %s: %w", coverDir, err)
	}

	// Find source files (non-test .py files) in the same directories as test files.
	sourceFiles := discoverSourceFiles(testFiles)

	var infos []CoverageInfo
	for _, sf := range sourceFiles {
		var info CoverageInfo
		if coll != nil {
			info = realCoverage(sf, coll)
		} else {
			info = estimateCoverage(sf, testFiles)
		}
		infos = append(infos, info)
	}

	// Write text report.
	reportPath := filepath.Join(coverDir, "coverage.txt")
	f, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("coverage: create report: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "File                          Lines  Covered  %")
	fmt.Fprintln(f, strings.Repeat("-", 55))
	var totalLines, totalCovered int
	for _, info := range infos {
		totalLines += info.TotalLines
		totalCovered += info.ExecLines
		fmt.Fprintf(f, "%-30s %5d  %7d  %5.1f%%\n",
			info.File, info.TotalLines, info.ExecLines, info.Coverage())
	}
	if totalLines > 0 {
		pct := float64(totalCovered) / float64(totalLines) * 100
		fmt.Fprintln(f, strings.Repeat("-", 55))
		fmt.Fprintf(f, "%-30s %5d  %7d  %5.1f%%\n", "Total", totalLines, totalCovered, pct)
	}

	fmt.Fprintf(stdout, "Coverage report written to %s\n", reportPath)
	return nil
}

func discoverSourceFiles(testFiles []string) []string {
	dirs := map[string]bool{}
	for _, tf := range testFiles {
		dirs[filepath.Dir(tf)] = true
	}
	var sources []string
	for dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".py") && !isTestFile(d.Name()) {
				sources = append(sources, path)
			}
			return nil
		})
	}
	return sources
}

// realCoverage builds CoverageInfo for sourceFile using actual hit data from coll.
func realCoverage(sourceFile string, coll *CoverageCollector) CoverageInfo {
	info := CoverageInfo{File: sourceFile}
	src, err := os.ReadFile(sourceFile)
	if err != nil {
		return info
	}
	coverable, err := CoverableLines(sourceFile, src)
	if err != nil {
		return info
	}
	info.TotalLines = len(coverable)
	hits := coll.HitsFor(sourceFile)
	for line := range coverable {
		if hits[line] {
			info.ExecLines++
		}
	}
	return info
}

func estimateCoverage(sourceFile string, testFiles []string) CoverageInfo {
	info := CoverageInfo{File: sourceFile}

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return info
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		info.TotalLines++
	}

	// A source file is "covered" if any test file in the same directory exists.
	base := strings.TrimSuffix(filepath.Base(sourceFile), ".py")
	dir := filepath.Dir(sourceFile)
	for _, tf := range testFiles {
		if filepath.Dir(tf) == dir {
			tfBase := filepath.Base(tf)
			if strings.Contains(tfBase, base) || strings.HasPrefix(tfBase, "test_") {
				// Estimate: assume test files cover ~70% of source lines.
				info.ExecLines = int(float64(info.TotalLines) * 0.70)
				break
			}
		}
	}
	return info
}
