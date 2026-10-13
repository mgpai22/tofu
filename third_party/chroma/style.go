package chroma

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Trilean uint8

const (
	Pass Trilean = iota
	Yes
	No
)

func (t Trilean) String() string {
	switch t {
	case Yes:
		return "Yes"
	case No:
		return "No"
	default:
		return "Pass"
	}
}

func (t Trilean) Prefix(s string) string {
	if t == Yes {
		return s
	} else if t == No {
		return "no" + s
	}
	return ""
}

type StyleEntry struct {
	Colour     Colour
	Background Colour
	Border     Colour

	Bold      Trilean
	Italic    Trilean
	Underline Trilean
	NoInherit bool
}

func (s StyleEntry) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s StyleEntry) String() string {
	out := []string{}
	if s.Bold != Pass {
		out = append(out, s.Bold.Prefix("bold"))
	}
	if s.Italic != Pass {
		out = append(out, s.Italic.Prefix("italic"))
	}
	if s.Underline != Pass {
		out = append(out, s.Underline.Prefix("underline"))
	}
	if s.NoInherit {
		out = append(out, "noinherit")
	}
	if s.Colour.IsSet() {
		out = append(out, s.Colour.String())
	}
	if s.Background.IsSet() {
		out = append(out, "bg:"+s.Background.String())
	}
	if s.Border.IsSet() {
		out = append(out, "border:"+s.Border.String())
	}
	return strings.Join(out, " ")
}

func (s StyleEntry) Sub(e StyleEntry) StyleEntry {
	out := StyleEntry{}
	if e.Colour != s.Colour {
		out.Colour = s.Colour
	}
	if e.Background != s.Background {
		out.Background = s.Background
	}
	if e.Bold != s.Bold {
		out.Bold = s.Bold
	}
	if e.Italic != s.Italic {
		out.Italic = s.Italic
	}
	if e.Underline != s.Underline {
		out.Underline = s.Underline
	}
	if e.Border != s.Border {
		out.Border = s.Border
	}
	return out
}

func (s StyleEntry) Inherit(ancestors ...StyleEntry) StyleEntry {
	out := s
	for i := len(ancestors) - 1; i >= 0; i-- {
		if out.NoInherit {
			return out
		}
		ancestor := ancestors[i]
		if !out.Colour.IsSet() {
			out.Colour = ancestor.Colour
		}
		if !out.Background.IsSet() {
			out.Background = ancestor.Background
		}
		if !out.Border.IsSet() {
			out.Border = ancestor.Border
		}
		if out.Bold == Pass {
			out.Bold = ancestor.Bold
		}
		if out.Italic == Pass {
			out.Italic = ancestor.Italic
		}
		if out.Underline == Pass {
			out.Underline = ancestor.Underline
		}
	}
	return out
}

func (s StyleEntry) IsZero() bool {
	return s.Colour == 0 && s.Background == 0 && s.Border == 0 && s.Bold == Pass && s.Italic == Pass &&
		s.Underline == Pass && !s.NoInherit
}

type StyleBuilder struct {
	entries map[TokenType]string
	name    string
	parent  *Style
}

func NewStyleBuilder(name string) *StyleBuilder {
	return &StyleBuilder{name: name, entries: map[TokenType]string{}}
}

func (s *StyleBuilder) AddAll(entries StyleEntries) *StyleBuilder {
	for ttype, entry := range entries {
		s.entries[ttype] = entry
	}
	return s
}

func (s *StyleBuilder) Get(ttype TokenType) StyleEntry {

	entry, _ := ParseStyleEntry(s.entries[ttype])
	if s.parent != nil {
		entry = entry.Inherit(s.parent.Get(ttype))
	}
	return entry
}

func (s *StyleBuilder) Add(ttype TokenType, entry string) *StyleBuilder {
	s.entries[ttype] = entry
	return s
}

func (s *StyleBuilder) AddEntry(ttype TokenType, entry StyleEntry) *StyleBuilder {
	s.entries[ttype] = entry.String()
	return s
}

func (s *StyleBuilder) Transform(transform func(StyleEntry) StyleEntry) *StyleBuilder {
	types := make(map[TokenType]struct{})
	for tt := range s.entries {
		types[tt] = struct{}{}
	}
	if s.parent != nil {
		for _, tt := range s.parent.Types() {
			types[tt] = struct{}{}
		}
	}
	for tt := range types {
		s.AddEntry(tt, transform(s.Get(tt)))
	}
	return s
}

func (s *StyleBuilder) Build() (*Style, error) {
	style := &Style{
		Name:    s.name,
		entries: map[TokenType]StyleEntry{},
		parent:  s.parent,
	}
	for ttype, descriptor := range s.entries {
		entry, err := ParseStyleEntry(descriptor)
		if err != nil {
			return nil, fmt.Errorf("invalid entry for %s: %s", ttype, err)
		}
		style.entries[ttype] = entry
	}
	return style, nil
}

type StyleEntries map[TokenType]string

func NewXMLStyle(r io.Reader) (*Style, error) {
	dec := xml.NewDecoder(r)
	style := &Style{}
	return style, dec.Decode(style)
}

func MustNewXMLStyle(r io.Reader) *Style {
	style, err := NewXMLStyle(r)
	if err != nil {
		panic(err)
	}
	return style
}

func NewStyle(name string, entries StyleEntries) (*Style, error) {
	return NewStyleBuilder(name).AddAll(entries).Build()
}

func MustNewStyle(name string, entries StyleEntries) *Style {
	style, err := NewStyle(name, entries)
	if err != nil {
		panic(err)
	}
	return style
}

type Style struct {
	Name    string
	entries map[TokenType]StyleEntry
	parent  *Style
}

func (s *Style) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if s.parent != nil {
		return fmt.Errorf("cannot marshal style with parent")
	}
	start.Name = xml.Name{Local: "style"}
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "name"}, Value: s.Name}}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	sorted := make([]TokenType, 0, len(s.entries))
	for ttype := range s.entries {
		sorted = append(sorted, ttype)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for _, ttype := range sorted {
		entry := s.entries[ttype]
		el := xml.StartElement{Name: xml.Name{Local: "entry"}}
		el.Attr = []xml.Attr{
			{Name: xml.Name{Local: "type"}, Value: ttype.String()},
			{Name: xml.Name{Local: "style"}, Value: entry.String()},
		}
		if err := e.EncodeToken(el); err != nil {
			return err
		}
		if err := e.EncodeToken(xml.EndElement{Name: el.Name}); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

func (s *Style) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "name" {
			s.Name = attr.Value
		} else {
			return fmt.Errorf("unexpected attribute %s", attr.Name.Local)
		}
	}
	if s.Name == "" {
		return fmt.Errorf("missing style name attribute")
	}
	s.entries = map[TokenType]StyleEntry{}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if el.Name.Local != "entry" {
				return fmt.Errorf("unexpected element %s", el.Name.Local)
			}
			var ttype TokenType
			var entry StyleEntry
			for _, attr := range el.Attr {
				switch attr.Name.Local {
				case "type":
					ttype, err = TokenTypeString(attr.Value)
					if err != nil {
						return err
					}

				case "style":
					entry, err = ParseStyleEntry(attr.Value)
					if err != nil {
						return err
					}

				default:
					return fmt.Errorf("unexpected attribute %s", attr.Name.Local)
				}
			}
			s.entries[ttype] = entry

		case xml.EndElement:
			if el.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (s *Style) Types() []TokenType {
	dedupe := map[TokenType]bool{}
	for tt := range s.entries {
		dedupe[tt] = true
	}
	if s.parent != nil {
		for _, tt := range s.parent.Types() {
			dedupe[tt] = true
		}
	}
	out := make([]TokenType, 0, len(dedupe))
	for tt := range dedupe {
		out = append(out, tt)
	}
	return out
}

func (s *Style) Builder() *StyleBuilder {
	return &StyleBuilder{
		name:    s.Name,
		entries: map[TokenType]string{},
		parent:  s,
	}
}

func (s *Style) Has(ttype TokenType) bool {
	return !s.get(ttype).IsZero() || s.synthesisable(ttype)
}

func (s *Style) Get(ttype TokenType) StyleEntry {
	return s.get(ttype).Inherit(
		s.get(Background),
		s.get(Text),
		s.get(ttype.Category()),
		s.get(ttype.SubCategory()))
}

func (s *Style) get(ttype TokenType) StyleEntry {
	out := s.entries[ttype]
	if out.IsZero() && s.parent != nil {
		return s.parent.get(ttype)
	}
	if out.IsZero() && s.synthesisable(ttype) {
		out = s.synthesise(ttype)
	}
	return out
}

func (s *Style) synthesise(ttype TokenType) StyleEntry {
	bg := s.get(Background)
	text := StyleEntry{Colour: bg.Colour}
	text.Colour = text.Colour.BrightenOrDarken(0.5)

	switch ttype {

	case LineHighlight:
		return StyleEntry{Background: bg.Background.BrightenOrDarken(0.1)}

	case LineNumbers, LineNumbersTable:
		return text

	default:
		return StyleEntry{}
	}
}

func (s *Style) synthesisable(ttype TokenType) bool {
	return ttype == LineHighlight || ttype == LineNumbers || ttype == LineNumbersTable
}

func MustParseStyleEntry(entry string) StyleEntry {
	out, err := ParseStyleEntry(entry)
	if err != nil {
		panic(err)
	}
	return out
}

func ParseStyleEntry(entry string) (StyleEntry, error) {
	out := StyleEntry{}
	parts := strings.Fields(entry)
	for _, part := range parts {
		switch {
		case part == "italic":
			out.Italic = Yes
		case part == "noitalic":
			out.Italic = No
		case part == "bold":
			out.Bold = Yes
		case part == "nobold":
			out.Bold = No
		case part == "underline":
			out.Underline = Yes
		case part == "nounderline":
			out.Underline = No
		case part == "inherit":
			out.NoInherit = false
		case part == "noinherit":
			out.NoInherit = true
		case part == "bg:":
			out.Background = 0
		case strings.HasPrefix(part, "bg:#"):
			out.Background = ParseColour(part[3:])
			if !out.Background.IsSet() {
				return StyleEntry{}, fmt.Errorf("invalid background colour %q", part)
			}
		case strings.HasPrefix(part, "border:#"):
			out.Border = ParseColour(part[7:])
			if !out.Border.IsSet() {
				return StyleEntry{}, fmt.Errorf("invalid border colour %q", part)
			}
		case strings.HasPrefix(part, "#"):
			out.Colour = ParseColour(part)
			if !out.Colour.IsSet() {
				return StyleEntry{}, fmt.Errorf("invalid colour %q", part)
			}
		default:
			return StyleEntry{}, fmt.Errorf("unknown style element %q", part)
		}
	}
	return out, nil
}
