package htmlutil

import (
	"github.com/jaytaylor/html2text"
)

// GenerateText returns text given the html
func GenerateText(html string) (string, error) {
	text, err := html2text.FromString(html, html2text.Options{PrettyTables: true})
	return text, err
}
