package ui

import (
	"strings"
	"testing"
)

func TestRenderStandupBody_HeadersAndBullets(t *testing.T) {
	in := "**Yesterday**\nShipped two PRs and reviewed checks.\n\n**Today**\n- Monitor CI\n- Start sprint review\n\n**Blockers**\nNone."
	out := renderStandupBody(in, 80)
	for _, want := range []string{"Yesterday", "Today", "Blockers", "Monitor CI", "None."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
	if strings.Contains(out, "**") {
		t.Errorf("bold markers not stripped:\n%s", out)
	}
}

func TestRenderStandupBody_NoHeaders(t *testing.T) {
	in := "Just a freeform paragraph without any sections at all."
	out := renderStandupBody(in, 80)
	if !strings.Contains(out, "freeform") {
		t.Errorf("body lost: %s", out)
	}
}

func TestMatchHeader(t *testing.T) {
	cases := map[string]string{
		"**Yesterday**": "Yesterday",
		"## Today":      "Today",
		"Blockers:":     "Blockers",
		"yesterday":     "Yesterday",
		"random text":   "",
	}
	for in, want := range cases {
		got, ok := matchHeader(in)
		if want == "" && ok {
			t.Errorf("matchHeader(%q) matched %q, expected no match", in, got)
		}
		if want != "" && got != want {
			t.Errorf("matchHeader(%q) = %q; want %q", in, got, want)
		}
	}
}
