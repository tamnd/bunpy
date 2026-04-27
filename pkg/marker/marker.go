// Package marker is a pure-Go evaluator for the PEP 508
// environment-marker subset bunpy needs to filter Requires-Dist
// edges during resolution.
//
// The grammar covers the expression after the `;` in a wheel's
// Requires-Dist line. Variables in scope: python_version,
// python_full_version, os_name, sys_platform, platform_machine,
// platform_system, implementation_name, implementation_version,
// extra. Operators: ==, !=, <, <=, >, >=, in, not in. Boolean
// combinators: and, or, not. Parentheses group.
//
// python_version and python_full_version compare as PEP 440
// versions; every other variable compares as a string. Unknown
// variable names evaluate as the empty string per PEP 508 sec 5.
// extra is parsed but always empty in v0.1.5 so optional-deps
// stay excluded today; v0.1.6's lane work fills it.
package marker

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"

	"github.com/tamnd/bunpy/v1/pkg/version"
)

// Env is the marker environment. Empty fields evaluate as
// the empty string per PEP 508 sec 5.
type Env struct {
	PythonVersion         string
	PythonFullVersion     string
	OSName                string
	SysPlatform           string
	PlatformMachine       string
	PlatformSystem        string
	ImplementationName    string
	ImplementationVersion string
	Extra                 string
}

// DefaultEnv builds a marker environment from the host runtime.
// Python version is hard-coded to the goipy embedded version
// until the runtime exposes its own version string.
func DefaultEnv() Env {
	return Env{
		PythonVersion:         "3.14",
		PythonFullVersion:     "3.14.0",
		OSName:                osName(runtime.GOOS),
		SysPlatform:           sysPlatform(runtime.GOOS),
		PlatformMachine:       platformMachine(runtime.GOARCH),
		PlatformSystem:        platformSystem(runtime.GOOS),
		ImplementationName:    "goipy",
		ImplementationVersion: "3.14.0",
	}
}

func osName(goos string) string {
	if goos == "windows" {
		return "nt"
	}
	return "posix"
}

func sysPlatform(goos string) string {
	switch goos {
	case "windows":
		return "win32"
	case "darwin":
		return "darwin"
	default:
		return goos
	}
}

func platformMachine(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i686"
	default:
		return goarch
	}
}

func platformSystem(goos string) string {
	switch goos {
	case "windows":
		return "Windows"
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	default:
		return strings.ToTitle(goos[:1]) + goos[1:]
	}
}

// Expr is one parsed marker expression.
type Expr interface {
	Eval(env Env) bool
	String() string
}

// Parse turns a marker source string into an Expr. An empty
// string parses as the always-true expression.
func Parse(s string) (Expr, error) {
	if strings.TrimSpace(s) == "" {
		return alwaysTrue{}, nil
	}
	tokens, err := lex(s)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkEOF {
		return nil, fmt.Errorf("marker: column %d: unexpected %q", p.peek().col, p.peek().text)
	}
	return expr, nil
}

type alwaysTrue struct{}

func (alwaysTrue) Eval(Env) bool  { return true }
func (alwaysTrue) String() string { return "" }

// --- AST nodes ---

type cmpExpr struct {
	left  operand
	op    string
	right operand
}

func (c *cmpExpr) Eval(env Env) bool {
	lv := c.left.value(env)
	rv := c.right.value(env)
	if c.left.isVersionVar() || c.right.isVersionVar() {
		switch c.op {
		case "==", "!=", "<", "<=", ">", ">=":
			cmp, ok := compareAsVersion(lv, rv)
			if ok {
				return matchOp(c.op, cmp)
			}
		}
	}
	switch c.op {
	case "==":
		return lv == rv
	case "!=":
		return lv != rv
	case "<":
		return lv < rv
	case "<=":
		return lv <= rv
	case ">":
		return lv > rv
	case ">=":
		return lv >= rv
	case "in":
		return strings.Contains(rv, lv)
	case "not in":
		return !strings.Contains(rv, lv)
	}
	return false
}

func (c *cmpExpr) String() string {
	return fmt.Sprintf("%s %s %s", c.left, c.op, c.right)
}

func compareAsVersion(a, b string) (int, bool) {
	if a == "" || b == "" {
		return 0, false
	}
	return version.Compare(a, b), true
}

func matchOp(op string, cmp int) bool {
	switch op {
	case "==":
		return cmp == 0
	case "!=":
		return cmp != 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	}
	return false
}

type andExpr struct{ left, right Expr }

func (a *andExpr) Eval(env Env) bool { return a.left.Eval(env) && a.right.Eval(env) }
func (a *andExpr) String() string    { return fmt.Sprintf("%s and %s", a.left, a.right) }

type orExpr struct{ left, right Expr }

func (o *orExpr) Eval(env Env) bool { return o.left.Eval(env) || o.right.Eval(env) }
func (o *orExpr) String() string    { return fmt.Sprintf("%s or %s", o.left, o.right) }

type notExpr struct{ inner Expr }

func (n *notExpr) Eval(env Env) bool { return !n.inner.Eval(env) }
func (n *notExpr) String() string    { return fmt.Sprintf("not %s", n.inner) }

// --- operands ---

type operand interface {
	value(env Env) string
	isVersionVar() bool
	String() string
}

type variable string

func (v variable) value(env Env) string {
	switch string(v) {
	case "python_version":
		return env.PythonVersion
	case "python_full_version":
		return env.PythonFullVersion
	case "os_name":
		return env.OSName
	case "sys_platform":
		return env.SysPlatform
	case "platform_machine":
		return env.PlatformMachine
	case "platform_system":
		return env.PlatformSystem
	case "implementation_name":
		return env.ImplementationName
	case "implementation_version":
		return env.ImplementationVersion
	case "extra":
		return env.Extra
	}
	return ""
}

func (v variable) isVersionVar() bool {
	return v == "python_version" || v == "python_full_version" || v == "implementation_version"
}

func (v variable) String() string { return string(v) }

type literal string

func (l literal) value(Env) string   { return string(l) }
func (l literal) isVersionVar() bool { return false }
func (l literal) String() string     { return fmt.Sprintf("%q", string(l)) }

// --- lexer ---

type tokenKind int

const (
	tkEOF tokenKind = iota
	tkIdent
	tkString
	tkOp
	tkLParen
	tkRParen
)

type token struct {
	kind tokenKind
	text string
	col  int
}

func lex(src string) ([]token, error) {
	var out []token
	i := 0
	for i < len(src) {
		r := rune(src[i])
		col := i + 1
		switch {
		case unicode.IsSpace(r):
			i++
			continue
		case r == '(':
			out = append(out, token{tkLParen, "(", col})
			i++
		case r == ')':
			out = append(out, token{tkRParen, ")", col})
			i++
		case r == '\'' || r == '"':
			quote := byte(r)
			j := i + 1
			for j < len(src) && src[j] != quote {
				j++
			}
			if j >= len(src) {
				return nil, fmt.Errorf("marker: column %d: unterminated string", col)
			}
			out = append(out, token{tkString, src[i+1 : j], col})
			i = j + 1
		case r == '=' || r == '!' || r == '<' || r == '>':
			if i+1 < len(src) && src[i+1] == '=' {
				out = append(out, token{tkOp, src[i : i+2], col})
				i += 2
			} else if r == '<' || r == '>' {
				out = append(out, token{tkOp, src[i : i+1], col})
				i++
			} else {
				return nil, fmt.Errorf("marker: column %d: unexpected %q", col, string(r))
			}
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(src) && (unicode.IsLetter(rune(src[j])) || unicode.IsDigit(rune(src[j])) || src[j] == '_' || src[j] == '.') {
				j++
			}
			word := src[i:j]
			if word == "in" || word == "not" || word == "and" || word == "or" {
				out = append(out, token{tkOp, word, col})
			} else {
				out = append(out, token{tkIdent, word, col})
			}
			i = j
		default:
			return nil, fmt.Errorf("marker: column %d: unexpected %q", col, string(r))
		}
	}
	out = append(out, token{tkEOF, "", len(src) + 1})
	return out, nil
}

// --- parser ---

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() token { return p.tokens[p.pos] }

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tkOp && p.peek().text == "or" {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &orExpr{left, right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tkOp && p.peek().text == "and" {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &andExpr{left, right}
	}
	return left, nil
}

func (p *parser) parseNot() (Expr, error) {
	if p.peek().kind == tkOp && p.peek().text == "not" {
		// `not in` is a binary op handled in parseCmp; bare `not` here.
		// Disambiguate: `not` followed by an operand+`in` is the binary op,
		// but our lexer emits `in` as a separate token, so we can peek.
		if p.pos+2 < len(p.tokens) {
			next := p.tokens[p.pos+1]
			if next.kind == tkIdent || next.kind == tkString || next.kind == tkLParen {
				// it's a unary `not <expr>`
				p.advance()
				inner, err := p.parseNot()
				if err != nil {
					return nil, err
				}
				return &notExpr{inner}, nil
			}
		}
	}
	return p.parseCmp()
}

func (p *parser) parseCmp() (Expr, error) {
	if p.peek().kind == tkLParen {
		p.advance()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("marker: column %d: expected ')' got %q", p.peek().col, p.peek().text)
		}
		p.advance()
		return expr, nil
	}
	left, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	op, err := p.parseOp()
	if err != nil {
		return nil, err
	}
	right, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	return &cmpExpr{left, op, right}, nil
}

func (p *parser) parseOperand() (operand, error) {
	t := p.advance()
	switch t.kind {
	case tkIdent:
		return variable(t.text), nil
	case tkString:
		return literal(t.text), nil
	}
	return nil, fmt.Errorf("marker: column %d: expected operand got %q", t.col, t.text)
}

func (p *parser) parseOp() (string, error) {
	t := p.advance()
	switch {
	case t.kind == tkOp && (t.text == "==" || t.text == "!=" || t.text == "<" || t.text == "<=" || t.text == ">" || t.text == ">="):
		return t.text, nil
	case t.kind == tkOp && t.text == "in":
		return "in", nil
	case t.kind == tkOp && t.text == "not":
		nt := p.advance()
		if nt.kind == tkOp && nt.text == "in" {
			return "not in", nil
		}
		return "", fmt.Errorf("marker: column %d: expected 'in' after 'not' got %q", nt.col, nt.text)
	}
	return "", fmt.Errorf("marker: column %d: expected operator got %q", t.col, t.text)
}
