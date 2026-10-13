package chroma

import (
	"fmt"
)

type Emitter interface {
	Emit(groups []string, state *LexerState) Iterator
}

type ValidatingEmitter interface {
	Emitter
	ValidateEmitter(rule *CompiledRule) error
}

type SerialisableEmitter interface {
	Emitter
	EmitterKind() string
}

type EmitterFunc func(groups []string, state *LexerState) Iterator

func (e EmitterFunc) Emit(groups []string, state *LexerState) Iterator {
	return e(groups, state)
}

type Emitters []Emitter

type byGroupsEmitter struct {
	Emitters
}

var _ ValidatingEmitter = (*byGroupsEmitter)(nil)

func ByGroups(emitters ...Emitter) Emitter {
	return &byGroupsEmitter{Emitters: emitters}
}

func (b *byGroupsEmitter) EmitterKind() string { return "bygroups" }

func (b *byGroupsEmitter) ValidateEmitter(rule *CompiledRule) error {
	if len(rule.Regexp.GetGroupNumbers())-1 != len(b.Emitters) {
		return fmt.Errorf("number of groups %d does not match number of emitters %d", len(rule.Regexp.GetGroupNumbers())-1, len(b.Emitters))
	}
	return nil
}

func (b *byGroupsEmitter) Emit(groups []string, state *LexerState) Iterator {
	iterators := make([]Iterator, 0, len(groups)-1)
	if len(b.Emitters) != len(groups)-1 {
		iterators = append(iterators, Error.Emit(groups, state))

	} else {
		for i, group := range groups[1:] {
			if b.Emitters[i] != nil {
				iterators = append(iterators, b.Emitters[i].Emit([]string{group}, state))
			}
		}
	}
	return Concaterator(iterators...)
}

func ByGroupNames(emitters map[string]Emitter) Emitter {
	return EmitterFunc(func(groups []string, state *LexerState) Iterator {
		iterators := make([]Iterator, 0, len(state.NamedGroups)-1)
		if len(state.NamedGroups)-1 == 0 {
			if emitter, ok := emitters[`0`]; ok {
				iterators = append(iterators, emitter.Emit(groups, state))
			} else {
				iterators = append(iterators, Error.Emit(groups, state))
			}
		} else {
			ruleRegex := state.Rules[state.State][state.Rule].Regexp
			for i := 1; i < len(state.NamedGroups); i++ {
				groupName := ruleRegex.GroupNameFromNumber(i)
				group := state.NamedGroups[groupName]
				if emitter, ok := emitters[groupName]; ok {
					if emitter != nil {
						iterators = append(iterators, emitter.Emit([]string{group}, state))
					}
				} else {
					iterators = append(iterators, Error.Emit([]string{group}, state))
				}
			}
		}
		return Concaterator(iterators...)
	})
}

func UsingByGroup(sublexerNameGroup, codeGroup int, emitters ...Emitter) Emitter {
	return &usingByGroup{
		SublexerNameGroup: sublexerNameGroup,
		CodeGroup:         codeGroup,
		Emitters:          emitters,
	}
}

type usingByGroup struct {
	SublexerNameGroup int      `xml:"sublexer_name_group"`
	CodeGroup         int      `xml:"code_group"`
	Emitters          Emitters `xml:"emitters"`
}

func (u *usingByGroup) EmitterKind() string { return "usingbygroup" }
func (u *usingByGroup) Emit(groups []string, state *LexerState) Iterator {

	if len(u.Emitters) != len(groups)-1 {
		panic("UsingByGroup expects number of emitters to be the same as len(groups)-1")
	}

	sublexer := state.Registry.Get(groups[u.SublexerNameGroup])

	iterators := make([]Iterator, len(groups)-1)
	for i, group := range groups[1:] {
		if i == u.CodeGroup-1 && sublexer != nil {
			var err error
			iterators[i], err = sublexer.Tokenise(nil, groups[u.CodeGroup])
			if err != nil {
				panic(err)
			}
		} else if u.Emitters[i] != nil {
			iterators[i] = u.Emitters[i].Emit([]string{group}, state)
		}
	}
	return Concaterator(iterators...)
}

func UsingLexer(lexer Lexer) Emitter {
	return EmitterFunc(func(groups []string, _ *LexerState) Iterator {
		it, err := lexer.Tokenise(&TokeniseOptions{State: "root", Nested: true}, groups[0])
		if err != nil {
			panic(err)
		}
		return it
	})
}

type usingEmitter struct {
	Lexer string `xml:"lexer,attr"`
}

func (u *usingEmitter) EmitterKind() string { return "using" }

func (u *usingEmitter) Emit(groups []string, state *LexerState) Iterator {
	if state.Registry == nil {
		panic(fmt.Sprintf("no LexerRegistry available for Using(%q)", u.Lexer))
	}
	lexer := state.Registry.Get(u.Lexer)
	if lexer == nil {
		panic(fmt.Sprintf("no such lexer %q", u.Lexer))
	}
	it, err := lexer.Tokenise(&TokeniseOptions{State: "root", Nested: true}, groups[0])
	if err != nil {
		panic(err)
	}
	return it
}

func Using(lexer string) Emitter {
	return &usingEmitter{Lexer: lexer}
}

type usingSelfEmitter struct {
	State string `xml:"state,attr"`
}

func (u *usingSelfEmitter) EmitterKind() string { return "usingself" }

func (u *usingSelfEmitter) Emit(groups []string, state *LexerState) Iterator {
	it, err := state.Lexer.Tokenise(&TokeniseOptions{State: u.State, Nested: true}, groups[0])
	if err != nil {
		panic(err)
	}
	return it
}

func UsingSelf(stateName string) Emitter {
	return &usingSelfEmitter{stateName}
}
