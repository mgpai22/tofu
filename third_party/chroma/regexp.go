package chroma

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

type Rule struct {
	Pattern string
	Type    Emitter
	Mutator Mutator
}

func Words(prefix, suffix string, words ...string) string {
	sort.Slice(words, func(i, j int) bool {
		return len(words[j]) < len(words[i])
	})
	for i, word := range words {
		words[i] = regexp.QuoteMeta(word)
	}
	return prefix + `(` + strings.Join(words, `|`) + `)` + suffix
}

func Tokenise(lexer Lexer, options *TokeniseOptions, text string) ([]Token, error) {
	var out []Token
	it, err := lexer.Tokenise(options, text)
	if err != nil {
		return nil, err
	}
	for t := it(); t != EOF; t = it() {
		out = append(out, t)
	}
	return out, nil
}

type Rules map[string][]Rule

func (r Rules) Rename(oldRule, newRule string) Rules {
	r = r.Clone()
	r[newRule] = r[oldRule]
	delete(r, oldRule)
	return r
}

func (r Rules) Clone() Rules {
	out := map[string][]Rule{}
	for key, rules := range r {
		out[key] = make([]Rule, len(rules))
		copy(out[key], rules)
	}
	return out
}

func (r Rules) Merge(rules Rules) Rules {
	out := r.Clone()
	for k, v := range rules.Clone() {
		out[k] = v
	}
	return out
}

func MustNewLexer(config *Config, rules func() Rules) *RegexLexer {
	lexer, err := NewLexer(config, rules)
	if err != nil {
		panic(err)
	}
	return lexer
}

func NewLexer(config *Config, rulesFunc func() Rules) (*RegexLexer, error) {
	if config == nil {
		config = &Config{}
	}
	for _, glob := range append(config.Filenames, config.AliasFilenames...) {
		_, err := filepath.Match(glob, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not a valid glob: %w", config.Name, glob, err)
		}
	}
	r := &RegexLexer{
		config:         config,
		fetchRulesFunc: func() (Rules, error) { return rulesFunc(), nil },
	}

	return r, nil
}

func (r *RegexLexer) Trace(trace bool) *RegexLexer {
	r.trace = trace
	return r
}

func (r *RegexLexer) SetTracing(trace bool) {
	r.trace = trace
}

type CompiledRule struct {
	Rule
	Regexp *regexp2.Regexp
	flags  string
}

type CompiledRules map[string][]*CompiledRule

type LexerState struct {
	Lexer    *RegexLexer
	Registry *LexerRegistry
	Text     []rune
	Pos      int
	Rules    CompiledRules
	Stack    []string
	State    string
	Rule     int

	Groups []string

	NamedGroups map[string]string

	MutatorContext map[interface{}]interface{}
	iteratorStack  []Iterator
	options        *TokeniseOptions
	newlineAdded   bool
}

func (l *LexerState) Set(key interface{}, value interface{}) {
	l.MutatorContext[key] = value
}

func (l *LexerState) Get(key interface{}) interface{} {
	return l.MutatorContext[key]
}

func (l *LexerState) Iterator() Token {
	trace := json.NewEncoder(os.Stderr)
	end := len(l.Text)
	if l.newlineAdded {
		end--
	}
	for l.Pos < end && len(l.Stack) > 0 {

		for len(l.iteratorStack) > 0 {
			n := len(l.iteratorStack) - 1
			t := l.iteratorStack[n]()
			if t.Type == Ignore {
				continue
			}
			if t == EOF {
				l.iteratorStack = l.iteratorStack[:n]
				continue
			}
			return t
		}

		l.State = l.Stack[len(l.Stack)-1]
		selectedRule, ok := l.Rules[l.State]
		if !ok {
			panic("unknown state " + l.State)
		}
		var start time.Time
		if l.Lexer.trace {
			start = time.Now()
		}
		ruleIndex, rule, groups, namedGroups := matchRules(l.Text, l.Pos, selectedRule)
		if l.Lexer.trace {
			var length int
			if groups != nil {
				length = len(groups[0])
			} else {
				length = -1
			}
			_ = trace.Encode(Trace{ //nolint
				Lexer:   l.Lexer.config.Name,
				State:   l.State,
				Rule:    ruleIndex,
				Pattern: rule.Pattern,
				Pos:     l.Pos,
				Length:  length,
				Elapsed: float64(time.Since(start)) / float64(time.Millisecond),
			})

		}

		if groups == nil {

			if l.Text[l.Pos] == '\n' && l.State != l.options.State {
				l.Stack = []string{l.options.State}
				continue
			}
			l.Pos++
			return Token{Error, string(l.Text[l.Pos-1 : l.Pos])}
		}
		l.Rule = ruleIndex
		l.Groups = groups
		l.NamedGroups = namedGroups
		l.Pos += utf8.RuneCountInString(groups[0])
		if rule.Mutator != nil {
			if err := rule.Mutator.Mutate(l); err != nil {
				panic(err)
			}
		}
		if rule.Type != nil {
			l.iteratorStack = append(l.iteratorStack, rule.Type.Emit(l.Groups, l))
		}
	}

	for len(l.iteratorStack) > 0 {
		n := len(l.iteratorStack) - 1
		t := l.iteratorStack[n]()
		if t.Type == Ignore {
			continue
		}
		if t == EOF {
			l.iteratorStack = l.iteratorStack[:n]
			continue
		}
		return t
	}

	if l.Pos != len(l.Text) && len(l.Stack) == 0 {
		value := string(l.Text[l.Pos:])
		l.Pos = len(l.Text)
		return Token{Type: Error, Value: value}
	}
	return EOF
}

type RegexLexer struct {
	registry *LexerRegistry
	config   *Config
	analyser func(text string) float32
	trace    bool

	mu             sync.Mutex
	compiled       bool
	rawRules       Rules
	rules          map[string][]*CompiledRule
	fetchRulesFunc func() (Rules, error)
	compileOnce    sync.Once
	compileError   error
}

func (r *RegexLexer) String() string {
	return r.config.Name
}

func (r *RegexLexer) Rules() (Rules, error) {
	if err := r.needRules(); err != nil {
		return nil, err
	}
	return r.rawRules, nil
}

func (r *RegexLexer) SetRegistry(registry *LexerRegistry) Lexer {
	r.registry = registry
	return r
}

func (r *RegexLexer) SetAnalyser(analyser func(text string) float32) Lexer {
	r.analyser = analyser
	return r
}

func (r *RegexLexer) AnalyseText(text string) float32 {
	if r.analyser != nil {
		return r.analyser(text)
	}
	return 0
}

func (r *RegexLexer) SetConfig(config *Config) *RegexLexer {
	r.config = config
	return r
}

func (r *RegexLexer) Config() *Config {
	return r.config
}

func (r *RegexLexer) maybeCompile() (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.compiled {
		return nil
	}
	for state, rules := range r.rules {
		for i, rule := range rules {
			if rule.Regexp == nil {
				pattern := "(?:" + rule.Pattern + ")"
				if rule.flags != "" {
					pattern = "(?" + rule.flags + ")" + pattern
				}
				pattern = `\G` + pattern
				rule.Regexp, err = regexp2.Compile(pattern, 0)
				if err != nil {
					return fmt.Errorf("failed to compile rule %s.%d: %s", state, i, err)
				}
				rule.Regexp.MatchTimeout = time.Millisecond * 250
			}
		}
	}
restart:
	seen := map[LexerMutator]bool{}
	for state := range r.rules {
		for i := range len(r.rules[state]) {
			rule := r.rules[state][i]
			if compile, ok := rule.Mutator.(LexerMutator); ok {
				if seen[compile] {
					return fmt.Errorf("saw mutator %T twice; this should not happen", compile)
				}
				seen[compile] = true
				if err := compile.MutateLexer(r.rules, state, i); err != nil {
					return err
				}

				goto restart
			}
		}
	}

	for state := range r.rules {
		for i := range len(r.rules[state]) {
			rule := r.rules[state][i]
			if validate, ok := rule.Type.(ValidatingEmitter); ok {
				if err := validate.ValidateEmitter(rule); err != nil {
					return fmt.Errorf("%s: %s: %s: %w", r.config.Name, state, rule.Pattern, err)
				}
			}
		}
	}
	r.compiled = true
	return nil
}

func (r *RegexLexer) fetchRules() error {
	rules, err := r.fetchRulesFunc()
	if err != nil {
		return fmt.Errorf("%s: failed to compile rules: %w", r.config.Name, err)
	}
	if _, ok := rules["root"]; !ok {
		return fmt.Errorf("no \"root\" state")
	}
	compiledRules := map[string][]*CompiledRule{}
	for state, rules := range rules {
		compiledRules[state] = nil
		for _, rule := range rules {
			flags := ""
			if !r.config.NotMultiline {
				flags += "m"
			}
			if r.config.CaseInsensitive {
				flags += "i"
			}
			if r.config.DotAll {
				flags += "s"
			}
			compiledRules[state] = append(compiledRules[state], &CompiledRule{Rule: rule, flags: flags})
		}
	}

	r.rawRules = rules
	r.rules = compiledRules
	return nil
}

func (r *RegexLexer) needRules() error {
	var err error
	if r.fetchRulesFunc != nil {
		r.compileOnce.Do(func() {
			r.compileError = r.fetchRules()
		})
		if r.compileError != nil {
			return r.compileError
		}
	}
	if err := r.maybeCompile(); err != nil {
		return err
	}
	return err
}

func (r *RegexLexer) Tokenise(options *TokeniseOptions, text string) (Iterator, error) {
	err := r.needRules()
	if err != nil {
		return nil, err
	}
	if options == nil {
		options = defaultOptions
	}
	if options.EnsureLF {
		text = ensureLF(text)
	}
	newlineAdded := false
	if !options.Nested && r.config.EnsureNL && !strings.HasSuffix(text, "\n") {
		text += "\n"
		newlineAdded = true
	}
	state := &LexerState{
		Registry:       r.registry,
		newlineAdded:   newlineAdded,
		options:        options,
		Lexer:          r,
		Text:           []rune(text),
		Stack:          []string{options.State},
		Rules:          r.rules,
		MutatorContext: map[interface{}]interface{}{},
	}
	return state.Iterator, nil
}

func (r *RegexLexer) MustRules() Rules {
	rules, err := r.Rules()
	if err != nil {
		panic(err)
	}
	return rules
}

func matchRules(text []rune, pos int, rules []*CompiledRule) (int, *CompiledRule, []string, map[string]string) {
	for i, rule := range rules {
		match, err := rule.Regexp.FindRunesMatchStartingAt(text, pos)
		if match != nil && err == nil && match.Index == pos {
			groups := []string{}
			namedGroups := make(map[string]string)
			for _, g := range match.Groups() {
				namedGroups[g.Name] = g.String()
				groups = append(groups, g.String())
			}
			return i, rule, groups, namedGroups
		}
	}
	return 0, &CompiledRule{}, nil, nil
}

func ensureLF(text string) string {
	buf := make([]byte, len(text))
	var j int
	for i := range len(text) {
		c := text[i]
		if c == '\r' {
			if i < len(text)-1 && text[i+1] == '\n' {
				continue
			}
			c = '\n'
		}
		buf[j] = c
		j++
	}
	return string(buf[:j])
}
