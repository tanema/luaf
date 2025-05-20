// Package luaf is an implementation of lua 5.4 for learning purposes and luafs 🤠.
// It aims to be fully feature compatible with lua 5.4 as well as additions to the
// standard library to make it more of an everyday use language instead of just
// as an embedded language.
//
//	`luaf` is still very WIP and really shouldn't be used by anyone except me and
//	maybe people who are interested in lua implementations.
//
//	`luaf` should be fully compatible with the lua APIs that are default in lua,
//	however it will not provide the same API as the C API. It will also be able to
//	precompile and run precompiled code however that precompiled code is not compatible
//	with `lua`. `luac` will not be able to run code from `luaf` and vise versa.
//	Since the point of this implementation is more for using lua than it's use in Go
//	there is less of an emphasis on a go API though a simple API exists.
package luaf

import (
	"context"
	"strings"

	"github.com/tanema/luaf/src/parse"
	"github.com/tanema/luaf/src/runtime"
)

// Env is a simple mapping for what will be exposed as a global in the lua source.
// This can be used to expose functions values as apis.
// The values in keys and values should only be
// - int64
// - float64
// - string
// - *runtime.GoFunc (use runtime.Fn() to create them easily)
// So this means that to add your own api, you can just create an env
//
//	env := map[any]any{
//			"render": runtime.Fn("render", func(vm *runtime.VM, args []any) ([]any, error) {
//				return nil, errors.New("not implemented")
//			},
//	}
type Env map[any]any

// String will simply parse and run lua source code. Label is a replacement for
// a filename so that it will be easier to debug.
func String(label, src string, env Env, args ...string) ([]any, error) {
	fn, err := parse.Parse(label, strings.NewReader(src), parse.ModeText)
	if err != nil {
		return nil, err
	}
	vm := runtime.New(context.Background(), runtime.NewTable(nil, env), args...)
	defer func() { _ = vm.Close() }()
	return vm.Eval(fn)
}

// File will parse and eval a lua source file.
func File(filepath string, env Env, args ...string) ([]any, error) {
	fn, err := parse.File(filepath, parse.ModeBinary&parse.ModeText)
	if err != nil {
		return nil, err
	}
	vm := runtime.New(context.Background(), runtime.NewTable(nil, env), args...)
	defer func() { _ = vm.Close() }()
	return vm.Eval(fn)
}
