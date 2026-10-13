package chroma

import (
	"fmt"
	"strings"
)

var (
	defaultOptions = &TokeniseOptions{
		State:    "root",
		EnsureLF: true,
	}
)

type Config struct {
	Name string `xml:"name,omitempty"`

	Aliases []string `xml:"alias,omitempty"`

	Filenames []string `xml:"filename,omitempty"`

	AliasFilenames []string `xml:"alias_filename,omitempty"`

	MimeTypes []string `xml:"mime_type,omitempty"`

	CaseInsensitive bool `xml:"case_insensitive,omitempty"`

	DotAll bool `xml:"dot_all,omitempty"`

	NotMultiline bool `xml:"not_multiline,omitempty"`

	EnsureNL bool `xml:"ensure_nl,omitempty"`

	Priority float32 `xml:"priority,omitempty"`

	Analyse *AnalyseConfig `xml:"analyse,omitempty"`
}

type AnalyseConfig struct {
	Regexes []RegexConfig `xml:"regex,omitempty"`

	First bool `xml:"first,attr"`
}

type RegexConfig struct {
	Pattern string  `xml:"pattern,attr"`
	Score   float32 `xml:"score,attr"`
}

type Token struct {
	Type  TokenType `json:"type"`
	Value string    `json:"value"`
}

func (t *Token) String() string   { return t.Value }
func (t *Token) GoString() string { return fmt.Sprintf("&Token{%s, %q}", t.Type, t.Value) }

func (t *Token) Clone() Token {
	return *t
}

var EOF Token

type TokeniseOptions struct {
	State string

	Nested bool

	EnsureLF bool
}

type Lexer interface {
	Config() *Config

	Tokenise(options *TokeniseOptions, text string) (Iterator, error)

	SetRegistry(registry *LexerRegistry) Lexer

	SetAnalyser(analyser func(text string) float32) Lexer

	AnalyseText(text string) float32
}

type Trace struct {
	Lexer   string  `json:"lexer"`
	State   string  `json:"state"`
	Rule    int     `json:"rule"`
	Pattern string  `json:"pattern"`
	Pos     int     `json:"pos"`
	Length  int     `json:"length"`
	Elapsed float64 `json:"elapsedMs"`
}

type TracingLexer interface {
	Lexer
	SetTracing(enable bool)
}

type Lexers []Lexer

func (l Lexers) Len() int      { return len(l) }
func (l Lexers) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l Lexers) Less(i, j int) bool {
	return strings.ToLower(l[i].Config().Name) < strings.ToLower(l[j].Config().Name)
}

type PrioritisedLexers []Lexer

func (l PrioritisedLexers) Len() int      { return len(l) }
func (l PrioritisedLexers) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l PrioritisedLexers) Less(i, j int) bool {
	ip := l[i].Config().Priority
	if ip == 0 {
		ip = 1
	}
	jp := l[j].Config().Priority
	if jp == 0 {
		jp = 1
	}
	return ip > jp
}

type Analyser interface {
	AnalyseText(text string) float32
}
