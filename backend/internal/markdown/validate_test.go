package markdown

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate_acceptsPlainGFM(t *testing.T) {
	md := `# Title

Paragraph with **bold** and a [link](https://example.com).

| a | b |
|---|---|
| 1 | 2 |

![fig](assets/diagram.png)
`
	if err := Validate(md); err != nil {
		t.Fatalf("expected valid markdown: %v", err)
	}
}

func TestValidate_rejectsRawHTML(t *testing.T) {
	cases := []string{
		"# Hi\n\n<div>oops</div>\n",
		"# Hi\n\n<script>alert(1)</script>\n",
		"text <em>x</em> more",
	}
	for _, c := range cases {
		err := Validate(c)
		if err == nil {
			t.Fatalf("expected error for %q", c)
		}
		var pe *PolicyError
		if !errors.As(err, &pe) || pe.Code != "raw_html" {
			t.Fatalf("expected raw_html PolicyError, got %v", err)
		}
	}
}

func TestValidate_rejectsBadSchemes(t *testing.T) {
	cases := []struct {
		src  string
		code string
	}{
		{"[x](javascript:alert(1))", "disallowed_scheme"},
		{"[x](data:text/html,base64)", "disallowed_scheme"},
		{"![x](data:image/png;base64,AAA)", "disallowed_scheme"},
		{"[x](file:///etc/passwd)", "disallowed_scheme"},
		{"[x](//evil.example)", "disallowed_url"},
		{"[x](relative.md)", "disallowed_url"},
	}
	for _, tc := range cases {
		err := Validate(tc.src)
		if err == nil {
			t.Fatalf("expected error for %q", tc.src)
		}
		var pe *PolicyError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PolicyError for %q, got %v", tc.src, err)
		}
		if pe.Code != tc.code {
			t.Fatalf("for %q: want code %s, got %s (%s)", tc.src, tc.code, pe.Code, pe.Detail)
		}
	}
}

func TestValidate_allowsMailtoAndFragment(t *testing.T) {
	if err := Validate("[e](mailto:a@b.co)"); err != nil {
		t.Fatal(err)
	}
	if err := Validate("[s](#section-1)"); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_contentTooLarge(t *testing.T) {
	err := validateWithMaxLen(strings.Repeat("a", 101), 100)
	var pe *PolicyError
	if !errors.As(err, &pe) || pe.Code != "content_too_large" {
		t.Fatalf("expected content_too_large, got %v", err)
	}
}
