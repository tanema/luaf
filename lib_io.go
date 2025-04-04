package luaf

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

var (
	stdin = &File{
		handle:    os.Stdin,
		reader:    bufio.NewReader(os.Stdin),
		path:      "<stdin>",
		readOnly:  true,
		isstdpipe: true,
	}
	stdout = &File{
		handle:    os.Stdout,
		path:      "<stdout>",
		writeOnly: true,
		isstdpipe: true,
	}
	stderr = &File{
		handle:    os.Stderr,
		path:      "<stderr>",
		writeOnly: true,
		isstdpipe: true,
	}
	defaultInput  = stdin
	defaultOutput = stdout
)

var fileMetatable *Table

func createIOLib() *Table {
	fileMetatable = &Table{
		hashtable: map[any]Value{
			"__name":     &String{val: "FILE*"},
			"__tostring": Fn("file:__tostring", stdIOFileString),
			"__close":    Fn("file:__close", stdIOFileClose),
			"__gc":       Fn("file:__gc", stdIOFileClose),
			"__index": &Table{
				hashtable: map[any]Value{
					"close":   Fn("file:close", stdIOFileClose),
					"flush":   Fn("file:flush", stdIOFileFlush),
					"read":    Fn("file:read", stdIOFileRead),
					"write":   Fn("file:write", stdIOFileWrite),
					"lines":   Fn("file:lines", stdIOFileLines),
					"seek":    Fn("file:seek", stdIOFileSeek),
					"setvbuf": Fn("file:setvbuf", stdIOFileSetvbuf),
				},
			},
		},
	}

	return &Table{
		hashtable: map[any]Value{
			"stderr":  stderr,
			"stdin":   stdin,
			"stdout":  stdout,
			"input":   Fn("io.input", stdIOInput),
			"output":  Fn("io.output", stdIOOutput),
			"open":    Fn("io.open", stdIOOpen),
			"close":   Fn("io.close", stdIOClose),
			"flush":   Fn("io.flush", stdIOFlush),
			"tmpfile": Fn("io.tmpfile", stdIOTmpfile),
			"type":    Fn("io.type", stdIOType),
			"read":    Fn("io.read", stdIORead),
			"write":   Fn("io.write", stdIOWrite),
			"lines":   Fn("io.lines", stdIOLines),
			"popen":   Fn("io.popen", stdIOPOpen),
		},
	}
}

func stdIOClose(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.close", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*File)
	}
	if err := file.Close(); err != nil {
		return []Value{&Boolean{val: false}, &String{val: fmt.Sprintf("problem closing file: %v", err.Error())}}, nil
	}
	return []Value{&Boolean{val: true}}, nil
}

func stdIOFileClose(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:close", "file"); err != nil {
		return nil, err
	}
	file, _ := args[0].(*File)
	if err := file.Close(); err != nil {
		return []Value{&Boolean{val: false}, &String{val: fmt.Sprintf("problem closing file: %v", err.Error())}}, nil
	}
	return []Value{&Boolean{val: true}}, nil
}

func stdIOFileString(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:__tostring", "file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*File)
	}
	return []Value{&String{val: file.String()}}, nil
}

func stdIOFlush(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.flush", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*File)
	}
	if err := file.handle.Sync(); err != nil {
		return nil, fmt.Errorf("problem flushing file: %v", err.Error())
	}
	return []Value{}, nil
}

func stdIOFileFlush(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:flush", "file"); err != nil {
		return nil, err
	}
	file, _ := args[0].(*File)
	if err := file.handle.Sync(); err != nil {
		return nil, fmt.Errorf("problem flushing file: %v", err.Error())
	}
	return []Value{}, nil
}

func stdIOOpen(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.open", "string", "~string"); err != nil {
		return nil, err
	}
	filepath, _ := args[0].(*String)
	mode := "r"
	if len(args) > 1 {
		mode = args[1].(*String).val
	}

	var filemode int
	var readOnly, writeOnly bool
	switch mode {
	case "r", "rb":
		filemode = os.O_RDONLY
		readOnly = true
	case "w", "wb":
		filemode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		writeOnly = true
	case "a", "ab":
		filemode = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		writeOnly = true
	case "r+", "rb+":
		filemode = os.O_RDWR
	case "w+", "wb+":
		filemode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	case "a+", "ab+":
		filemode = os.O_RDWR | os.O_CREATE | os.O_APPEND
	default:
		return nil, argumentErr(2, "io.open", fmt.Errorf("invalid filemode %q", mode))
	}
	file, err := NewFile(filepath.val, filemode, readOnly, writeOnly)
	var retVals []Value
	if err != nil {
		retVals = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
	} else {
		retVals = []Value{file}
	}
	return retVals, nil
}

func stdIOTmpfile(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.tmpfile"); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp("", "")
	var retVals []Value
	if err != nil {
		retVals = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
	} else {
		newFile := &File{
			handle: file,
			path:   file.Name(),
			reader: bufio.NewReader(file),
		}
		retVals = []Value{newFile}
	}
	return retVals, nil
}

func stdIOType(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.input", "value"); err != nil {
		return nil, err
	}
	switch f := args[0].(type) {
	case *File:
		if f.closed {
			return []Value{&String{val: "closed file"}}, nil
		}
		return []Value{&String{val: "file"}}, nil
	default:
		return []Value{&Nil{}}, nil
	}
}

func stdIOInput(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.input", "~file|string"); err != nil {
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
		file, err = NewFile(farg.val, os.O_RDWR, false, false)
		if err != nil {
			return nil, fmt.Errorf("cannot set default input (%s)", err.Error())
		}
	}
	defaultInput = file
	return []Value{}, nil
}

func stdIOOutput(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.output", "~file|string"); err != nil {
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
		file, err = NewFile(farg.val, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, false, true)
		if err != nil {
			return nil, fmt.Errorf("cannot set default output (%s)", err.Error())
		}
	}
	defaultOutput = file
	return []Value{defaultOutput}, nil
}

func stdIOWrite(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.write"); err != nil {
		return nil, err
	}
	return defaultOutput.Write(vm, args)
}

func stdIOFileWrite(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:write", "file"); err != nil {
		return nil, err
	}
	return args[0].(*File).Write(vm, args[1:])
}

func stdIORead(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.read", "~file", "~string"); err != nil {
		return nil, err
	}
	file := defaultInput
	if len(args) > 0 {
		file = args[0].(*File)
		if f, isFile := args[0].(*File); isFile {
			file = f
			args = args[1:]
		}
	}
	return file.Read(vm, args)
}

func stdIOFileRead(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:read", "file", "~string"); err != nil {
		return nil, err
	}
	return args[0].(*File).Read(vm, args[1:])
}

func stdIOLinesNext(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.lines.next", "file"); err != nil {
		return nil, err
	}
	file := args[0].(*File)
	text, err := file.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return []Value{&Nil{}}, nil
		}
		return nil, fmt.Errorf("problem reading file: %w", err)
	}
	return []Value{&String{val: text}}, nil
}

func stdIOLines(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.lines", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file = args[0].(*File)
	}
	return []Value{Fn("io.lines.next", stdIOLinesNext), file, &Nil{}}, nil
}

func stdIOFileLines(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:lines", "file"); err != nil {
		return nil, err
	}
	return []Value{Fn("file:lines.next", stdIOLinesNext), args[0].(*File), &Nil{}}, nil
}

func stdIOFileSeek(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "file:seek", "file", "~string", "~number"); err != nil {
		return nil, err
	}
	file := args[0].(*File)
	if file.closed {
		return nil, argumentErr(1, "file:seek", errors.New("file closed"))
	} else if file.process != nil {
		return nil, argumentErr(1, "file:seek", errors.New("cannot seek process"))
	}
	whence := 1
	if len(args) > 1 {
		switch args[1].(*String).val {
		case "set":
			whence = 0
		case "cur":
			whence = 1
		case "end":
			whence = 2
		}
	}
	offset := int64(0)
	if len(args) > 2 {
		offset = toInt(args[2])
	}

	pos, err := file.handle.Seek(offset, whence)
	var retVals []Value
	if err != nil {
		retVals = []Value{&Nil{}, &String{val: err.Error()}}
	} else {
		retVals = []Value{&Integer{val: pos}}
	}
	return retVals, nil
}

func stdIOFileSetvbuf(*VM, []Value) ([]Value, error) {
	// not supported.
	return []Value{&Boolean{val: true}}, nil
}

func popenCommand(arg string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("C:\\Windows\\system32\\cmd.exe", append([]string{"/c"}, arg)...)
	}
	return exec.Command("/bin/sh", append([]string{"-c"}, arg)...)
}

func stdIOPOpen(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "io.popen", "string", "~string"); err != nil {
		return nil, err
	}
	cmd := popenCommand(args[0].(*String).val)
	mode := "r"
	if len(args) > 1 {
		mode = args[1].(*String).val
	}

	newFile := &File{path: args[0].(*String).val}
	if mode == "r" {
		stderr, _ := cmd.StderrPipe()
		stdout, _ := cmd.StdoutPipe()
		newFile.reader = bufio.NewReader(io.MultiReader(stdout, stderr))
		newFile.readOnly = true
	} else if mode == "w" {
		stdin, _ := cmd.StdinPipe()
		newFile.handle = writerCloserToFile(stdin)
		newFile.writeOnly = true
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	newFile.process = cmd.Process
	return []Value{newFile}, nil
}
