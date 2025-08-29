package runtime

import (
	"errors"
	"fmt"

	"github.com/tanema/luaf/src/lerrors"
	"github.com/tanema/luaf/src/parse"
)

func newUserErr(vm *VM, level int, val any) error {
	var ci callInfo
	if csl := len(vm.callStack); csl > 0 && level > 0 && level < csl {
		ci = vm.callStack[level]
	} else {
		ci = vm.callStack[csl-1]
	}
	parts := []string{}
	for i := range vm.callDepth {
		info := vm.callStack[i]
		parts = append(parts, fmt.Sprintf("\t%v:%v: in %v", info.filename, info.Line, info.name))
	}

	var err error
	if str, isStr := val.(string); isStr {
		err = errors.New(str)
	} else {
		err = fmt.Errorf("(error object is a %v value)", typeName(val))
	}

	return &lerrors.Error{
		Kind:      lerrors.UserErr,
		Filename:  ci.filename,
		Line:      ci.Line,
		Column:    ci.Column,
		Err:       err,
		Traceback: parts,
		Value:     val,
	}
}

func newRuntimeErr(vm *VM, li parse.LineInfo, err error) error {
	var luaErr *lerrors.Error
	if errors.As(err, &luaErr) {
		return luaErr
	}
	ci := callInfo{LineInfo: li}
	if len(vm.callStack) > 0 {
		ci.filename = vm.callStack[len(vm.callStack)-1].filename
	}
	parts := []string{}
	for i := range vm.callDepth {
		info := vm.callStack[i]
		parts = append(parts, fmt.Sprintf("\t%v:%v: in %v", info.filename, info.Line, info.name))
	}
	return &lerrors.Error{
		Kind:      lerrors.RuntimeErr,
		Filename:  ci.filename,
		Line:      ci.Line,
		Column:    ci.Column,
		Err:       err,
		Traceback: parts,
	}
}
