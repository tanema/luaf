// Package lerrors are a unified errors package for lua parsing and runtime so
// that they can be formatted in a unified way and handled in a unified way.
package lerrors

import (
	"fmt"
	"strings"
)

type (
	// ErrorKind is an enum to describe where the error originates from.
	ErrorKind int
	// Error captures all errors in the luaf runtime. It distinguishes between lexer, parser
	// runtime, and user errors and will format them accordingly. This is so that
	// errors can be handled in a uniform way in the runtime.
	Error struct {
		Line      int64
		Column    int64
		Kind      ErrorKind
		Err       error
		Filename  string
		Traceback []string
	}
)

const (
	// RuntimeErr is an error that originates from the runtime.
	RuntimeErr ErrorKind = iota
	// ParserErr is an error that originates from the parser.
	ParserErr
	// LexerErr is an error that originates from the lexer.
	LexerErr
	// UserErr is an error raised from user code by the user.
	UserErr
)

func (err *Error) Error() string {
	switch err.Kind {
	case RuntimeErr:
		return fmt.Sprintf(
			"lua:%v:%v:%v %v\nstack traceback:\n%v",
			err.Filename,
			err.Line,
			err.Column,
			err.Err,
			strings.Join(err.Traceback, "\n"),
		)
	case ParserErr:
		return fmt.Sprintf(`Parse Error: %s:%v:%v %v`, err.Filename, err.Line, err.Column, err.Err)
	case LexerErr:
		return fmt.Sprintf("Lex Error: %v", err.Err.Error())
	default:
		return err.Err.Error()
	}
}
