package chroma

import (
	"path/filepath"
	"sort"
	"strings"
)

var (
	ignoredSuffixes = [...]string{

		"~", ".bak", ".old", ".orig",

		".dpkg-dist", ".dpkg-old", ".ucf-dist", ".ucf-new", ".ucf-old",

		".rpmnew", ".rpmorig", ".rpmsave",

		".in",
	}
)

type LexerRegistry struct {
	Lexers  Lexers
	byName  map[string]Lexer
	byAlias map[string]Lexer
}

func NewLexerRegistry() *LexerRegistry {
	return &LexerRegistry{
		byName:  map[string]Lexer{},
		byAlias: map[string]Lexer{},
	}
}

func (l *LexerRegistry) Names(withAliases bool) []string {
	out := []string{}
	for _, lexer := range l.Lexers {
		config := lexer.Config()
		out = append(out, config.Name)
		if withAliases {
			out = append(out, config.Aliases...)
		}
	}
	sort.Strings(out)
	return out
}

func (l *LexerRegistry) Aliases(skipWithoutAliases bool) []string {
	out := []string{}
	for _, lexer := range l.Lexers {
		config := lexer.Config()
		if len(config.Aliases) == 0 {
			if skipWithoutAliases {
				continue
			}
			out = append(out, config.Name)
		}
		out = append(out, config.Aliases...)
	}
	sort.Strings(out)
	return out
}

func (l *LexerRegistry) Get(name string) Lexer {
	if lexer := l.byName[name]; lexer != nil {
		return lexer
	}
	if lexer := l.byAlias[name]; lexer != nil {
		return lexer
	}
	if lexer := l.byName[strings.ToLower(name)]; lexer != nil {
		return lexer
	}
	if lexer := l.byAlias[strings.ToLower(name)]; lexer != nil {
		return lexer
	}

	candidates := PrioritisedLexers{}

	if lexer := l.Match("filename." + name); lexer != nil {
		candidates = append(candidates, lexer)
	}

	if lexer := l.Match(name); lexer != nil {
		candidates = append(candidates, lexer)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Sort(candidates)
	return candidates[0]
}

func (l *LexerRegistry) MatchMimeType(mimeType string) Lexer {
	matched := PrioritisedLexers{}
	for _, l := range l.Lexers {
		for _, lmt := range l.Config().MimeTypes {
			if mimeType == lmt {
				matched = append(matched, l)
			}
		}
	}
	if len(matched) != 0 {
		sort.Sort(matched)
		return matched[0]
	}
	return nil
}

func (l *LexerRegistry) Match(filename string) Lexer {
	filename = filepath.Base(filename)
	matched := PrioritisedLexers{}

	for _, lexer := range l.Lexers {
		config := lexer.Config()
		for _, glob := range config.Filenames {
			ok, err := filepath.Match(glob, filename)
			if err != nil {
				panic(err)
			} else if ok {
				matched = append(matched, lexer)
			} else {
				for _, suf := range &ignoredSuffixes {
					ok, err := filepath.Match(glob+suf, filename)
					if err != nil {
						panic(err)
					} else if ok {
						matched = append(matched, lexer)
						break
					}
				}
			}
		}
	}
	if len(matched) > 0 {
		sort.Sort(matched)
		return matched[0]
	}
	matched = nil

	for _, lexer := range l.Lexers {
		config := lexer.Config()
		for _, glob := range config.AliasFilenames {
			ok, err := filepath.Match(glob, filename)
			if err != nil {
				panic(err)
			} else if ok {
				matched = append(matched, lexer)
			} else {
				for _, suf := range &ignoredSuffixes {
					ok, err := filepath.Match(glob+suf, filename)
					if err != nil {
						panic(err)
					} else if ok {
						matched = append(matched, lexer)
						break
					}
				}
			}
		}
	}
	if len(matched) > 0 {
		sort.Sort(matched)
		return matched[0]
	}
	return nil
}

func (l *LexerRegistry) Analyse(text string) Lexer {
	var picked Lexer
	highest := float32(0.0)
	for _, lexer := range l.Lexers {
		if analyser, ok := lexer.(Analyser); ok {
			weight := analyser.AnalyseText(text)
			if weight > highest {
				picked = lexer
				highest = weight
			}
		}
	}
	return picked
}

func (l *LexerRegistry) Register(lexer Lexer) Lexer {
	lexer.SetRegistry(l)
	config := lexer.Config()

	l.byName[config.Name] = lexer
	l.byName[strings.ToLower(config.Name)] = lexer

	for _, alias := range config.Aliases {
		l.byAlias[alias] = lexer
		l.byAlias[strings.ToLower(alias)] = lexer
	}

	l.Lexers = add(l.Lexers, lexer)

	return lexer
}

func add(lexers Lexers, lexer Lexer) Lexers {
	for i, val := range lexers {
		if val == nil {
			continue
		}

		if val.Config().Name == lexer.Config().Name {
			lexers[i] = lexer
			return lexers
		}
	}

	return append(lexers, lexer)
}
