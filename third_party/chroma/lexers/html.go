package lexers

import (
	"github.com/alecthomas/chroma/v2"
)

var HTML = chroma.MustNewXMLLexer(embedded, "embedded/html.xml")
