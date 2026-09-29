package markdown

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MaxMarkdownBytes is the maximum allowed UTF-8 size for a single version body.
const MaxMarkdownBytes = 10 << 20

const maxMailtoLen = 4096

// PolicyError describes a rejected SOP markdown document. Use errors.As to detect it.
type PolicyError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *PolicyError) Error() string {
	return fmt.Sprintf("markdown policy violation (%s): %s", e.Code, e.Detail)
}

// Validate checks markdown against the SOP authoring policy (no raw HTML, restricted URLs).
func Validate(content string) error {
	return validateWithMaxLen(content, MaxMarkdownBytes)
}

func validateWithMaxLen(content string, maxBytes int) error {
	if len(content) > maxBytes {
		return &PolicyError{
			Code:   "content_too_large",
			Detail: fmt.Sprintf("content exceeds maximum of %d bytes", maxBytes),
		}
	}

	src := []byte(content)
	md := AuthoringMarkdown()
	doc := md.Parser().Parse(text.NewReader(src))

	return ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.HTMLBlock:
			return ast.WalkStop, &PolicyError{
				Code:   "raw_html",
				Detail: "HTML blocks are not allowed in SOP markdown",
			}
		case *ast.RawHTML:
			return ast.WalkStop, &PolicyError{
				Code:   "raw_html",
				Detail: "inline HTML is not allowed in SOP markdown",
			}
		case *ast.Link:
			if err := checkDestinationBytes(n.Destination); err != nil {
				return ast.WalkStop, err
			}
		case *ast.Image:
			if err := checkDestinationBytes(n.Destination); err != nil {
				return ast.WalkStop, err
			}
		case *ast.LinkReferenceDefinition:
			if err := checkDestinationBytes(n.Destination); err != nil {
				return ast.WalkStop, err
			}
		case *ast.AutoLink:
			if err := checkDestinationBytes(n.URL(src)); err != nil {
				return ast.WalkStop, err
			}
		}
		return ast.WalkContinue, nil
	})
}

func checkDestinationBytes(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return checkDestination(string(b))
}

func checkDestination(raw string) error {
	d := strings.TrimSpace(raw)
	if d == "" {
		return nil
	}

	if strings.HasPrefix(d, "#") {
		return nil
	}

	lower := strings.ToLower(d)
	if strings.HasPrefix(lower, "assets/") {
		return nil
	}

	if strings.HasPrefix(lower, "mailto:") {
		if len(d) > maxMailtoLen {
			return &PolicyError{Code: "url_too_long", Detail: "mailto link exceeds maximum length"}
		}
		return nil
	}

	if strings.HasPrefix(d, "//") {
		return &PolicyError{Code: "disallowed_url", Detail: "protocol-relative URLs are not allowed"}
	}

	u, err := url.Parse(d)
	if err != nil {
		return &PolicyError{Code: "invalid_url", Detail: err.Error()}
	}

	if u.Scheme == "" {
		return &PolicyError{
			Code:   "disallowed_url",
			Detail: "relative URLs are not allowed (use https://, http://, mailto:, or assets/...)",
		}
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return &PolicyError{Code: "disallowed_url", Detail: "http(s) URL is missing a host"}
		}
		return nil
	default:
		return &PolicyError{
			Code:   "disallowed_scheme",
			Detail: u.Scheme,
		}
	}
}
