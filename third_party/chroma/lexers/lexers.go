package lexers

import (
	"embed"
	"io/fs"
	"sync"
	"sync/atomic"

	"github.com/alecthomas/chroma/v2"
)

//go:embed embedded
var embedded embed.FS

var (
	GlobalLexerRegistry = chroma.NewLexerRegistry()
	declaredBeforeBuild []chroma.Lexer
	registryBuilt       atomic.Bool
	registryOnce        sync.Once
)

func registry() *chroma.LexerRegistry {
	registryOnce.Do(func() {
		paths, err := fs.Glob(embedded, "embedded/*.xml")
		if err != nil {
			panic(err)
		}
		for _, path := range paths {
			GlobalLexerRegistry.Register(chroma.MustNewXMLLexer(embedded, path))
		}
		for _, lexer := range declaredBeforeBuild {
			GlobalLexerRegistry.Register(lexer)
		}
		declaredBeforeBuild = nil
		analyseDNS()
		analyseMySQL()
		analyseZed()
		registryBuilt.Store(true)
	})
	return GlobalLexerRegistry
}

func Names(withAliases bool) []string {
	return registry().Names(withAliases)
}

func Aliases(skipWithoutAliases bool) []string {
	return registry().Aliases(skipWithoutAliases)
}

func Get(name string) chroma.Lexer {
	return registry().Get(name)
}

func MatchMimeType(mimeType string) chroma.Lexer {
	return registry().MatchMimeType(mimeType)
}

func Match(filename string) chroma.Lexer {
	return registry().Match(filename)
}

func Register(lexer chroma.Lexer) chroma.Lexer {
	if registryBuilt.Load() {
		return GlobalLexerRegistry.Register(lexer)
	}
	declaredBeforeBuild = append(declaredBeforeBuild, lexer)
	return lexer
}

func Analyse(text string) chroma.Lexer {
	return registry().Analyse(text)
}

func PlaintextRules() chroma.Rules {
	return chroma.Rules{
		"root": []chroma.Rule{
			{`.+`, chroma.Text, nil},
			{`\n`, chroma.Text, nil},
		},
	}
}

var Fallback chroma.Lexer = chroma.MustNewLexer(&chroma.Config{
	Name:      "fallback",
	Filenames: []string{"*"},
	Priority:  -1,
}, PlaintextRules)
