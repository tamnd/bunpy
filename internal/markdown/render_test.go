package markdown_test

import (
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/markdown"
)

func TestRenderH1(t *testing.T) {
	out := markdown.Render("# Title\n")
	if !strings.Contains(out, "Title") {
		t.Error("H1 text missing")
	}
	// Bold + underline codes present.
	if !strings.Contains(out, "\033[1m") || !strings.Contains(out, "\033[4m") {
		t.Error("H1 should contain bold+underline ANSI codes")
	}
}

func TestRenderH2(t *testing.T) {
	out := markdown.Render("## Section\n")
	if !strings.Contains(out, "Section") {
		t.Error("H2 text missing")
	}
	if !strings.Contains(out, "\033[1m") {
		t.Error("H2 should contain bold code")
	}
}

func TestRenderBold(t *testing.T) {
	out := markdown.Render("**hello**\n")
	if strings.Contains(out, "**") {
		t.Error("** markers should be stripped")
	}
	if !strings.Contains(out, "hello") {
		t.Error("bold text missing")
	}
	if !strings.Contains(out, "\033[1m") {
		t.Error("bold ANSI code missing")
	}
}

func TestRenderItalic(t *testing.T) {
	out := markdown.Render("_italic_\n")
	if strings.Contains(out, "_") {
		t.Error("_ markers should be stripped")
	}
	if !strings.Contains(out, "italic") {
		t.Error("italic text missing")
	}
}

func TestRenderCode(t *testing.T) {
	out := markdown.Render("`code`\n")
	if strings.Contains(out, "`") {
		t.Error("backtick markers should be stripped")
	}
	if !strings.Contains(out, "code") {
		t.Error("code text missing")
	}
	if !strings.Contains(out, "\033[36m") {
		t.Error("cyan code missing")
	}
}

func TestRenderBullet(t *testing.T) {
	out := markdown.Render("- item one\n")
	if !strings.Contains(out, "•") {
		t.Error("bullet marker • missing")
	}
	if !strings.Contains(out, "item one") {
		t.Error("bullet text missing")
	}
}

func TestRenderBlockquote(t *testing.T) {
	out := markdown.Render("> quote text\n")
	if !strings.Contains(out, "│") {
		t.Error("blockquote │ marker missing")
	}
	if !strings.Contains(out, "quote text") {
		t.Error("blockquote text missing")
	}
}

func TestRenderPlain(t *testing.T) {
	out := markdown.Render("plain text\n")
	if !strings.Contains(out, "plain text") {
		t.Error("plain text missing")
	}
}

func TestRenderCodeFence(t *testing.T) {
	src := "```\nsome code\n```\n"
	out := markdown.Render(src)
	if strings.Contains(out, "```") {
		t.Error("code fence markers should be stripped")
	}
	if !strings.Contains(out, "some code") {
		t.Error("fenced code text missing")
	}
}
