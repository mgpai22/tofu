package formatters

import (
	"io"
	"sort"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/formatters/svg"
)

var (
	NoOp = Register("noop", chroma.FormatterFunc(func(w io.Writer, s *chroma.Style, iterator chroma.Iterator) error {
		for t := iterator(); t != chroma.EOF; t = iterator() {
			if _, err := io.WriteString(w, t.Value); err != nil {
				return err
			}
		}
		return nil
	}))

	htmlFull = Register("html", html.New(html.Standalone(true), html.WithClasses(true)))
	SVG      = Register("svg", svg.New(svg.EmbedFont("Liberation Mono", svg.FontLiberationMono, svg.WOFF)))
)

var Fallback = NoOp

var Registry = map[string]chroma.Formatter{}

func Names() []string {
	out := []string{}
	for name := range Registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func Get(name string) chroma.Formatter {
	if f, ok := Registry[name]; ok {
		return f
	}
	return Fallback
}

func Register(name string, formatter chroma.Formatter) chroma.Formatter {
	Registry[name] = formatter
	return formatter
}
