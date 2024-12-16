package luaf

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	stdin = &File{
		handle: os.Stdin,
		Reader: os.Stdin,
		path:   "stdin",
		closed: true, // cannot close stdin
	}
	stdout = &File{
		handle: os.Stdout,
		Writer: os.Stdout,
		path:   "stdout",
		closed: true, // cannot close stdout
	}
	stderr = &File{
		handle: os.Stderr,
		Writer: os.Stderr,
		path:   "stderr",
		closed: true, // cannot close stderr
	}
	defaultInput = &File{
		handle: os.Stdin,
		Reader: os.Stdin,
		path:   "stdin",
		closed: true, // cannot close stdin
	}
	defaultOutput = &File{
		handle: os.Stdout,
		Writer: os.Stdout,
		path:   "stdout",
		closed: true, // cannot close stdout
	}
)

var libIO = &Table{
	hashtable: map[any]Value{
		"stderr":  stderr,
		"stdin":   stdin,
		"stdout":  stdout,
		"input":   &ExternFunc{stdIOInput},
		"output":  &ExternFunc{stdIOOutput},
		"open":    &ExternFunc{stdIOOpen},
		"close":   &ExternFunc{stdIOClose},
		"flush":   &ExternFunc{stdIOFlush},
		"tmpfile": &ExternFunc{stdIOTmpfile},
		"type":    &ExternFunc{stdIOType},
		"read":    &ExternFunc{stdIORead},
		"write":   &ExternFunc{stdIOWrite},
		"lines":   &ExternFunc{},
		"popen":   &ExternFunc{},
	},
}

var fileMetatable = &Table{
	hashtable: map[any]Value{
		"__name":     &String{val: "FILE*"},
		"__tostring": &ExternFunc{stdIOFileString},
		"__close":    &ExternFunc{stdIOClose},
		"__gc":       &ExternFunc{stdIOClose},
		"__index": &Table{
			hashtable: map[any]Value{
				"close":   &ExternFunc{stdIOClose},
				"flush":   &ExternFunc{stdIOFlush},
				"read":    &ExternFunc{},
				"write":   &ExternFunc{stdIOWrite},
				"lines":   &ExternFunc{},
				"seek":    &ExternFunc{},
				"setvbuf": &ExternFunc{},
			},
		},
	},
}

func stdIOClose(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.close", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file = args[0].(*File)
	}
	if err := file.Close(); err != nil {
		return nil, vm.err("problem closing file: %v", err.Error())
	}
	return []Value{}, nil
}

func stdIOFileString(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "file:__tostring", "file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file = args[0].(*File)
	}
	return []Value{&String{val: file.String()}}, nil
}

func stdIOFlush(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.flush", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file = args[0].(*File)
	}
	if err := file.handle.Sync(); err != nil {
		return nil, vm.err("problem flushing file: %v", err.Error())
	}
	return []Value{}, nil
}

func stdIOOpen(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.input", "string", "~string"); err != nil {
		return nil, err
	}
	filepath := args[0].(*String).val
	mode := "r"
	if len(args) > 1 {
		mode = args[1].(*String).val
	}

	var filemode int
	switch mode {
	case "r", "rb":
		filemode = os.O_RDONLY
	case "w", "wb":
		filemode = os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	case "a", "ab":
		filemode = os.O_WRONLY | os.O_APPEND | os.O_CREATE
	case "r+", "rb+":
		filemode = os.O_RDWR
	case "w+", "wb+":
		filemode = os.O_RDWR | os.O_TRUNC | os.O_CREATE
	case "a+", "ab+":
		filemode = os.O_APPEND | os.O_RDWR | os.O_CREATE
	default:
		return nil, argumentErr(vm, 2, "io.open", fmt.Errorf("invalid filemode %q", mode))
	}
	file, err := NewFile(filepath, os.FileMode(filemode))
	if err != nil {
		return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
	}
	return []Value{file}, nil
}

func stdIOTmpfile(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.tmpfile"); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp("", "")
	if err != nil {
		return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
	}
	newFile := &File{
		handle: file,
		Reader: file,
		Writer: file,
		path:   file.Name(),
	}
	return []Value{newFile}, nil
}

func stdIOType(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.input", "value"); err != nil {
		return nil, err
	}
	switch f := args[0].(type) {
	case *File:
		if f.closed {
			return []Value{&String{val: "file"}}, nil
		}
		return []Value{&String{val: "closed file"}}, nil
	default:
		return []Value{&Nil{}}, nil
	}
}

func stdIOInput(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.input", "~file|string"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []Value{defaultInput}, nil
	}
	var file *File
	var err error
	switch farg := args[0].(type) {
	case *File:
		file = farg
	case *String:
		file, err = NewFile(farg.val, os.FileMode(os.O_RDWR))
		if err != nil {
			return nil, vm.err("cannot set default input (%s)", err.Error())
		}
	}
	defaultInput = file
	return []Value{}, nil
}

func stdIOOutput(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.output", "~file|string"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []Value{defaultOutput}, nil
	}
	var file *File
	var err error
	switch farg := args[0].(type) {
	case *File:
		file = farg
	case *String:
		file, err = NewFile(farg.val, os.FileMode(os.O_RDWR|os.O_TRUNC|os.O_CREATE))
		if err != nil {
			return nil, vm.err("cannot set default output (%s)", err.Error())
		}
	}
	defaultOutput = file
	return []Value{}, nil
}

func stdIOWrite(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.close"); err != nil {
		return nil, err
	}
	file := defaultOutput
	offset := 0
	if len(args) > 0 {
		if f, isFile := args[0].(*File); isFile {
			file = f
			offset = 1
		}
	}
	params := args[offset:]
	strParts := make([]string, len(params))
	for i, arg := range params {
		str, err := toString(vm, arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str.val
	}
	if _, err := fmt.Fprint(file.Writer, strings.Join(strParts, "")); err != nil {
		return nil, err
	}
	return []Value{file}, nil
}

func stdIORead(vm *VM, args []Value) ([]Value, error) {
	return nil, nil
}

func stdIOPOpen(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.popen", "string", "~string"); err != nil {
		return nil, err
	}
	userCmd := strings.Split(args[0].(*String).val, " ")
	cmd := exec.Command(userCmd[0], userCmd[1:]...)
	mode := "r"
	if len(args) > 1 {
		mode = args[1].(*String).val
	}

	newFile := &File{path: args[0].(*String).val}
	if mode == "r" {
		stderr, _ := cmd.StderrPipe()
		stdout, _ := cmd.StdoutPipe()
		newFile.Reader = io.MultiReader(stdout, stderr)
	} else if mode == "w" {
		stdin, _ := cmd.StdinPipe()
		newFile.Writer = stdin
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	newFile.process = cmd.Process
	return []Value{newFile}, nil
}
