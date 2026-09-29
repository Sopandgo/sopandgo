package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// AuthoringMarkdown returns the goldmark configuration used for SOP authoring
// validation and for PDF HTML generation so behavior stays aligned.
func AuthoringMarkdown() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM))
}
