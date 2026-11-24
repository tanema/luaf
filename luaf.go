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
	"io"
	"os"
	"strings"

	"github.com/tanema/luaf/internal/conf"
	"github.com/tanema/luaf/internal/parse"
	"github.com/tanema/luaf/internal/runtime"
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
//	env := map[string]any{
//			"render": runtime.Fn("render", func(vm *runtime.VM, args []any) ([]any, error) {
//				return nil, errors.New("not implemented")
//			},
//	}

// String is a wrapper for Eval that will wrap the string in a reader and evaluate it.
// The label provided will help you identify errors better, think of it as a
// filename.
func String(label, src string, cfg conf.Config, args ...string) ([]any, error) {
	return Eval(label, strings.NewReader(src), cfg, args...)
}

// File is a simple wrapper for Eval that will open the file, eval, then close the file.
func File(filepath string, cfg conf.Config, args ...string) ([]any, error) {
	src, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()
	return Eval(filepath, src, cfg, args...)
}

// ConfigFile will run a lua source file as a config file. Pass a config object
// and it will allow the developer to change config settings via lua. Once the
// method returns, the changes to the config are applied to the original struct.
//
//	cfg := Config{}
//	err := luaf.ConfigFile("cfg.lua", &cfg)
func ConfigFile(filename string, dst any) error {
	return runtime.ConfigFile(filename, dst)
}

// Eval will parse the source and evaluate it with the env provided so that you can
// expose values to the vm.
func Eval(label string, src io.ReadSeeker, cfg conf.Config, args ...string) ([]any, error) {
	fn, err := parse.Parse(label, src, parse.ModeText, cfg)
	if err != nil {
		return nil, err
	}
	vm, err := runtime.New(context.Background(), cfg, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = vm.Close() }()
	return vm.Eval(fn)
}
