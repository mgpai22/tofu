package chroma

import "strings"

type Iterator func() Token

func (i Iterator) Tokens() []Token {
	var out []Token
	for t := i(); t != EOF; t = i() {
		out = append(out, t)
	}
	return out
}

func (i Iterator) Stdlib() func(yield func(Token) bool) {
	return func(yield func(Token) bool) {
		for t := i(); t != EOF; t = i() {
			if !yield(t) {
				return
			}
		}
	}
}

func Concaterator(iterators ...Iterator) Iterator {
	return func() Token {
		for len(iterators) > 0 {
			t := iterators[0]()
			if t != EOF {
				return t
			}
			iterators = iterators[1:]
		}
		return EOF
	}
}

func Literator(tokens ...Token) Iterator {
	return func() Token {
		if len(tokens) == 0 {
			return EOF
		}
		token := tokens[0]
		tokens = tokens[1:]
		return token
	}
}

func SplitTokensIntoLines(tokens []Token) (out [][]Token) {
	var line []Token
tokenLoop:
	for _, token := range tokens {
		for strings.Contains(token.Value, "\n") {
			parts := strings.SplitAfterN(token.Value, "\n", 2)

			token.Value = parts[1]

			clone := token.Clone()
			clone.Value = parts[0]
			line = append(line, clone)
			out = append(out, line)
			line = nil

			if len(token.Value) == 0 {
				continue tokenLoop
			}
		}
		line = append(line, token)
	}
	if len(line) > 0 {
		out = append(out, line)
	}

	if len(out) > 0 {
		last := out[len(out)-1]
		if len(last) == 1 && last[0].Value == "" {
			out = out[:len(out)-1]
		}
	}
	return out
}
