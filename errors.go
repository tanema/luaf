package luaf

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type (
	errorKind int
	// Error captures all errors in the luaf runtime. It distinguishes between lexer, parser
	// runtime, and user errors and will format them accordingly. This is so that
	// errors can be handled in a uniform way in the runtime.
	Error struct {
		lineInfo
		kind      errorKind
		err       error
		filename  string
		val       any
		traceback []string
	}
)

const (
	runtimeErr errorKind = iota
	parserErr
	lexerErr
	userErr
)

func (err *Error) Error() string {
	switch err.kind {
	case runtimeErr:
		return fmt.Sprintf(
			"lua:%v:%v:%v %v\nstack traceback:\n%v",
			err.filename,
			err.Line,
			err.Column,
			err.err,
			strings.Join(err.traceback, "\n"),
		)
	case parserErr:
		return fmt.Sprintf(`Parse Error: %s:%v:%v %v`, err.filename, err.Line, err.Column, err.err)
	case lexerErr:
		return fmt.Sprintf("Lex Error: %v", err.err.Error())
	default:
		if str, isStr := err.val.(string); isStr {
			return str
		}
		return fmt.Sprintf("(error object is a %v value)", typeName(err.val))
	}
}

func newUserErr(vm *VM, level int, val any) error {
	var ci callInfo
	if csl := len(vm.callStack); csl > 0 && level > 0 && level < csl {
		ci = vm.callStack[level]
	} else {
		ci = vm.callStack[csl-1]
	}
	parts := []string{}
	for i := range vm.callDepth {
		parts = append(parts, fmt.Sprintf("\t%v", vm.callStack[i]))
	}
	return &Error{
		filename:  ci.filename,
		lineInfo:  ci.lineInfo,
		val:       val,
		traceback: parts,
	}
}

func newRuntimeErr(vm *VM, li lineInfo, err error) error {
	var luaErr *Error
	if errors.As(err, &luaErr) {
		return luaErr
	}
	ci := callInfo{lineInfo: li}
	if len(vm.callStack) > 0 {
		ci.filename = vm.callStack[len(vm.callStack)-1].filename
	}
	parts := []string{}
	for i := range vm.callDepth {
		parts = append(parts, fmt.Sprintf("\t%v", vm.callStack[i]))
	}
	return &Error{
		filename:  ci.filename,
		lineInfo:  li,
		err:       err,
		traceback: parts,
	}
}
