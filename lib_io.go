package luaf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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
		"lines":   &ExternFunc{stdIOLines},
		"popen":   &ExternFunc{stdIOPOpen},
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
				"read":    &ExternFunc{stdIORead},
				"write":   &ExternFunc{stdIOWrite},
				"lines":   &ExternFunc{stdIOLines},
				"seek":    &ExternFunc{stdIOFileSeek},
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
		path:   file.Name(),
		reader: bufio.NewReader(file),
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
			return []Value{&String{val: "closed file"}}, nil
		}
		return []Value{&String{val: "file"}}, nil
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
	if len(args) > 0 {
		if f, isFile := args[0].(*File); isFile {
			file = f
			args = args[1:]
		}
	}
	if file.closed {
		return nil, argumentErr(vm, 1, "io.write", fmt.Errorf("file closed"))
	} else if file.readOnly {
		return nil, argumentErr(vm, 1, "io.write", fmt.Errorf("file readonly"))
	}
	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := toString(vm, arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str.val
	}
	_, err := fmt.Fprintln(file.handle, strings.Join(strParts, ""))
	if err != nil {
		return nil, err
	}
	return []Value{file}, nil
}

func stdIORead(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.read", "~file", "~string"); err != nil {
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
	if file.closed {
		return nil, argumentErr(vm, 1, "io.read", fmt.Errorf("file closed"))
	} else if file.writeOnly {
		return nil, argumentErr(vm, 1, "io.read", fmt.Errorf("file writeonly"))
	}

	formats := []Value{&String{val: "l"}}
	if len(args) > 0 {
		formats = args
	}
	results := []Value{}
	for _, mode := range formats {
		switch fmode := mode.(type) {
		case *Integer, *Float:
			size := toInt(fmode)
			if size == 0 {
				_, err := file.reader.ReadByte()
				if err == io.EOF {
					results = append(results, &Nil{})
					return results, nil
				} else if err := file.reader.UnreadByte(); err != nil {
					return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
				}
				results = append(results, &String{})
				continue
			}
			buf := make([]byte, size)
			_, err := io.ReadFull(file.reader, buf)
			if err == io.EOF {
				results = append(results, &Nil{})
				return results, nil
			} else if err != nil {
				return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
			}
			results = append(results, &String{val: string(buf)})
		case *String:
			switch fmode.val {
			case "n":
				var v float64
				_, err := fmt.Fscanf(file.reader, "%f", &v)
				if err == io.EOF {
					results = append(results, &Nil{})
					return results, nil
				} else if err != nil {
					return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
				}
				results = append(results, &Float{val: v})
			case "a":
				buf, err := io.ReadAll(file.handle)
				if err == io.EOF {
					results = append(results, &String{})
					return results, nil
				} else if err != nil {
					return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
				}
				results = append(results, &String{val: string(buf)})
			case "l", "L":
				text, err := file.reader.ReadString('\n')
				if err != nil {
					return []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}, nil
				} else if fmode.val == "L" {
					results = append(results, &String{val: string(text)})
				} else {
					results = append(results, &String{val: strings.TrimRight(string(text), "\r\n")})
				}
			default:
				return []Value{&Nil{}, &String{val: fmt.Sprintf("unknown read mode %s", fmode.val)}, &Integer{val: 1}}, nil
			}
		default:
			return []Value{&Nil{}, &String{val: fmt.Sprintf("unknown read mode %s", mode.String())}, &Integer{val: 1}}, nil
		}
	}
	return results, nil
}

func stdIOLines(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "io.lines", "~file"); err != nil {
		return nil, err
	}
	// file := defaultOutput
	// if len(args) > 0 {
	//	file = args[0].(*File)
	// }
	return nil, nil
}

// Sets and gets the file position, measured from the beginning of the file, to the position given by offset plus a base specified by the string whence, as follows:
// "set": base is position 0 (beginning of the file);
// "cur": base is current position;
// "end": base is end of file;
// In case of success, seek returns the final file position, measured in bytes from the beginning of the file. If seek fails, it returns nil, plus a string describing the error.
// The default value for whence is "cur", and for offset is 0. Therefore, the call file:seek() returns the current file position, without changing it; the call file:seek("set") sets the position to the beginning of the file (and returns 0); and the call file:seek("end") sets the position to the end of the file, and returns its size.
func stdIOFileSeek(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(vm, args, "file:seek", "file", "~string", "~number"); err != nil {
		return nil, err
	}
	file := args[0].(*File)
	if file.closed {
		return nil, argumentErr(vm, 1, "file:seek", fmt.Errorf("file closed"))
	} else if file.process != nil {
		return nil, argumentErr(vm, 1, "file:seek", fmt.Errorf("cannot seek process"))
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
	if err != nil {
		return []Value{&Nil{}, &String{val: err.Error()}}, nil
	}
	return []Value{&Integer{val: pos}}, nil
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
