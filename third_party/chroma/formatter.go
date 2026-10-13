package chroma

import (
	"io"
)

type Formatter interface {
	Format(w io.Writer, style *Style, iterator Iterator) error
}

type FormatterFunc func(w io.Writer, style *Style, iterator Iterator) error

func (f FormatterFunc) Format(w io.Writer, s *Style, it Iterator) (err error) {
	defer func() {
		if perr := recover(); perr != nil {
			err = perr.(error)
		}
	}()
	return f(w, s, it)
}

type recoveringFormatter struct {
	Formatter
}

func (r recoveringFormatter) Format(w io.Writer, s *Style, it Iterator) (err error) {
	defer func() {
		if perr := recover(); perr != nil {
			err = perr.(error)
		}
	}()
	return r.Formatter.Format(w, s, it)
}

func RecoveringFormatter(formatter Formatter) Formatter { return recoveringFormatter{formatter} }
