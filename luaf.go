package luaf

import (
	"context"
	"strings"

	"github.com/tanema/luaf/src/parse"
	"github.com/tanema/luaf/src/runtime"
)

// String will simply parse and run lua source code.
func String(label, src string, args ...string) ([]any, error) {
	fn, err := parse.Parse(label, strings.NewReader(src), parse.ModeText)
	if err != nil {
		return nil, err
	}
	vm := runtime.NewVM(context.Background(), args...)
	defer func() { _ = vm.Close() }()
	return vm.Eval(fn)
}

// File will parse and eval a lua source file.
func File(filepath string, args ...string) ([]any, error) {
	fn, err := parse.File(filepath, parse.ModeBinary&parse.ModeText)
	if err != nil {
		return nil, err
	}
	vm := runtime.NewVM(context.Background(), args...)
	defer func() { _ = vm.Close() }()
	return vm.Eval(fn)
}
