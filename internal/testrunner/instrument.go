package testrunner

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	gocopyCompiler "github.com/tamnd/gocopy/compiler"
	gocopyMarshal "github.com/tamnd/gocopy/marshal"
	parser2 "github.com/tamnd/gopapy/parser"
)

// CoverageCollector records which (file, line) pairs were executed.
// It is safe for concurrent use.
type CoverageCollector struct {
	mu   sync.Mutex
	hits map[string]map[int]bool
}

// Record marks (file, line) as executed.
func (c *CoverageCollector) Record(file string, line int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hits == nil {
		c.hits = make(map[string]map[int]bool)
	}
	if c.hits[file] == nil {
		c.hits[file] = make(map[int]bool)
	}
	c.hits[file][line] = true
}

// HitsFor returns a snapshot of executed lines for file.
func (c *CoverageCollector) HitsFor(file string) map[int]bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	src := c.hits[file]
	if src == nil {
		return nil
	}
	out := make(map[int]bool, len(src))
	for k := range src {
		out[k] = true
	}
	return out
}

// CoverableLines parses src with the gopapy v0.6 parser and returns the set
// of 1-indexed line numbers that contain executable statements.
func CoverableLines(filename string, src []byte) (map[int]bool, error) {
	mod, err := parser2.ParseFile(filename, string(src))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	lines := make(map[int]bool)
	collectStmtLines(mod.Body, lines)
	return lines, nil
}

func collectStmtLines(stmts []parser2.Stmt, lines map[int]bool) {
	for _, s := range stmts {
		switch v := s.(type) {
		case *parser2.ExprStmt:
			lines[v.P.Line] = true
		case *parser2.Assign:
			lines[v.P.Line] = true
		case *parser2.AugAssign:
			lines[v.P.Line] = true
		case *parser2.AnnAssign:
			lines[v.P.Line] = true
		case *parser2.Return:
			lines[v.P.Line] = true
		case *parser2.Raise:
			lines[v.P.Line] = true
		case *parser2.Pass:
			lines[v.P.Line] = true
		case *parser2.Break:
			lines[v.P.Line] = true
		case *parser2.Continue:
			lines[v.P.Line] = true
		case *parser2.Import:
			lines[v.P.Line] = true
		case *parser2.ImportFrom:
			lines[v.P.Line] = true
		case *parser2.Global:
			lines[v.P.Line] = true
		case *parser2.Nonlocal:
			lines[v.P.Line] = true
		case *parser2.Delete:
			lines[v.P.Line] = true
		case *parser2.Assert:
			lines[v.P.Line] = true
		case *parser2.TypeAlias:
			lines[v.P.Line] = true
		case *parser2.If:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
			collectStmtLines(v.Orelse, lines)
		case *parser2.While:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
			collectStmtLines(v.Orelse, lines)
		case *parser2.For:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
			collectStmtLines(v.Orelse, lines)
		case *parser2.AsyncFor:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
			collectStmtLines(v.Orelse, lines)
		case *parser2.Try:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
			for _, h := range v.Handlers {
				lines[h.P.Line] = true
				collectStmtLines(h.Body, lines)
			}
			collectStmtLines(v.Orelse, lines)
			collectStmtLines(v.Finalbody, lines)
		case *parser2.TryStar:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
			for _, h := range v.Handlers {
				lines[h.P.Line] = true
				collectStmtLines(h.Body, lines)
			}
			collectStmtLines(v.Orelse, lines)
			collectStmtLines(v.Finalbody, lines)
		case *parser2.With:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
		case *parser2.AsyncWith:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
		case *parser2.FunctionDef:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
		case *parser2.AsyncFunctionDef:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
		case *parser2.ClassDef:
			lines[v.P.Line] = true
			collectStmtLines(v.Body, lines)
		case *parser2.Match:
			lines[v.P.Line] = true
			for _, mc := range v.Cases {
				lines[mc.P.Line] = true
				collectStmtLines(mc.Body, lines)
			}
		}
	}
}

// InjectHits rewrites src by inserting a `__cov_hit__(file, line)` call
// before each statement line listed in stmtLines. The injected call uses the
// same indentation as the original line so that it stays syntactically valid.
func InjectHits(filename string, src []byte, stmtLines map[int]bool) []byte {
	srcLines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(srcLines)+len(stmtLines))
	for i, line := range srcLines {
		lineno := i + 1
		if stmtLines[lineno] {
			indent := leadingWhitespace(line)
			out = append(out, fmt.Sprintf("%s__cov_hit__(%q, %d)", indent, filename, lineno))
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

func leadingWhitespace(s string) string {
	for i, ch := range s {
		if ch != ' ' && ch != '\t' {
			return s[:i]
		}
	}
	return s
}

// Instrument parses src, identifies coverable lines, and returns a rewritten
// copy with __cov_hit__ calls injected before each executable statement.
func Instrument(filename string, src []byte) ([]byte, error) {
	lines, err := CoverableLines(filename, src)
	if err != nil {
		return nil, err
	}
	return InjectHits(filename, src, lines), nil
}

// goipyMagic314 mirrors goipy/marshal.Magic314.
var goipyMagic314 = [4]byte{0x2b, 0x0e, 0x0d, 0x0a}

// CompileInstrumented compiles instrumented src and wraps the result in the
// 16-byte goipy .pyc header (magic + 4-byte flags + 8-byte validation).
func CompileInstrumented(src []byte, filename string) ([]byte, error) {
	co, err := gocopyCompiler.Compile(src, gocopyCompiler.Options{Filename: filename})
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", filename, err)
	}
	stream, err := gocopyMarshal.Marshal(co)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", filename, err)
	}
	var buf bytes.Buffer
	buf.Write(goipyMagic314[:])
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // flags
	_ = binary.Write(&buf, binary.LittleEndian, uint64(0)) // validation
	buf.Write(stream)
	return buf.Bytes(), nil
}
