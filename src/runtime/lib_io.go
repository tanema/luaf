package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tanema/luaf/src/lfile"
)

var (
	defaultInput  = lfile.Stdin
	defaultOutput = lfile.Stdout
)

var fileMetatable *Table

func createIOLib() *Table {
	fileMetatable = &Table{
		hashtable: map[any]any{
			"__name":     "FILE*",
			"__tostring": Fn("file:__tostring", stdIOFileString),
			"__close":    Fn("file:__close", stdIOFileClose),
			"__gc":       Fn("file:__gc", stdIOFileClose),
			"__index": &Table{
				hashtable: map[any]any{
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
		hashtable: map[any]any{
			"stderr":  lfile.Stderr,
			"stdin":   lfile.Stdin,
			"stdout":  lfile.Stdout,
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

func stdIOClose(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.close", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*lfile.File)
	}
	if err := file.Close(); err != nil {
		return []any{false, fmt.Sprintf("problem closing file: %v", err.Error())}, nil
	}
	return []any{true}, nil
}

func stdIOFileClose(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:close", "file"); err != nil {
		return nil, err
	}
	file, _ := args[0].(*lfile.File)
	if err := file.Close(); err != nil {
		return []any{false, fmt.Sprintf("problem closing file: %v", err.Error())}, nil
	}
	return []any{true}, nil
}

func stdIOFileString(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:__tostring", "file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*lfile.File)
	}
	return []any{ToString(file)}, nil
}

func stdIOFlush(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.flush", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*lfile.File)
	}
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("problem flushing file: %v", err.Error())
	}
	return []any{}, nil
}

func stdIOFileFlush(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:flush", "file"); err != nil {
		return nil, err
	}
	file, _ := args[0].(*lfile.File)
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("problem flushing file: %v", err.Error())
	}
	return []any{}, nil
}

func stdIOOpen(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.open", "string", "~string"); err != nil {
		return nil, err
	}
	filepath, _ := args[0].(string)
	mode := "r"
	if len(args) > 1 {
		mode = args[1].(string)
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
	file, err := lfile.Open(filepath, filemode, readOnly, writeOnly)
	var retVals []any
	if err != nil {
		retVals = []any{nil, err.Error(), int64(1)}
	} else {
		retVals = []any{file}
	}
	return retVals, nil
}

func stdIOTmpfile(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.tmpfile"); err != nil {
		return nil, err
	}
	newFile, err := lfile.CreateTmp()
	return []any{newFile}, err
}

func stdIOType(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.input", "value"); err != nil {
		return nil, err
	}
	switch f := args[0].(type) {
	case *lfile.File:
		if f.Closed {
			return []any{"closed file"}, nil
		}
		return []any{"file"}, nil
	default:
		return []any{nil}, nil
	}
}

func stdIOInput(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.input", "~file|string"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []any{defaultInput}, nil
	}
	var file *lfile.File
	var err error
	switch farg := args[0].(type) {
	case *lfile.File:
		file = farg
	case string:
		file, err = lfile.Open(farg, os.O_RDWR, false, false)
		if err != nil {
			return nil, fmt.Errorf("cannot set default input (%s)", err.Error())
		}
	}
	defaultInput = file
	return []any{}, nil
}

func stdIOOutput(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.output", "~file|string"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []any{defaultOutput}, nil
	}
	var file *lfile.File
	var err error
	switch farg := args[0].(type) {
	case *lfile.File:
		file = farg
	case string:
		file, err = lfile.Open(farg, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, false, true)
		if err != nil {
			return nil, fmt.Errorf("cannot set default output (%s)", err.Error())
		}
	}
	defaultOutput = file
	return []any{defaultOutput}, nil
}

func stdIOWrite(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.write"); err != nil {
		return nil, err
	}
	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := vm.toString(arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str
	}
	return []any{}, defaultOutput.Write(strings.Join(strParts, ""))
}

func stdIOFileWrite(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:write", "file"); err != nil {
		return nil, err
	}

	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := vm.toString(arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str
	}
	return []any{}, args[0].(*lfile.File).Write(strings.Join(strParts, ""))
}

func stdIORead(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.read", "~file", "~string"); err != nil {
		return nil, err
	}
	file := defaultInput
	if len(args) > 0 {
		file = args[0].(*lfile.File)
		if f, isFile := args[0].(*lfile.File); isFile {
			file = f
			args = args[1:]
		}
	}

	formats := []any{"l"}
	if len(args) > 0 {
		formats = args
	}
	return file.Read(formats)
}

func stdIOFileRead(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:read", "file", "~string"); err != nil {
		return nil, err
	}
	formats := []any{"l"}
	if len(args) > 0 {
		formats = args
	}
	return args[0].(*lfile.File).Read(formats)
}

func stdIOLinesNext(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.lines.next", "file"); err != nil {
		return nil, err
	}
	file := args[0].(*lfile.File)
	text, err := file.Read([]any{"l"})
	if err != nil {
		if errors.Is(err, io.EOF) {
			return []any{nil}, nil
		}
		return nil, fmt.Errorf("problem reading file: %w", err)
	}
	return []any{text}, nil
}

func stdIOLines(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.lines", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file = args[0].(*lfile.File)
	}
	return []any{Fn("io.lines.next", stdIOLinesNext), file, nil}, nil
}

func stdIOFileLines(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:lines", "file"); err != nil {
		return nil, err
	}
	return []any{Fn("file:lines.next", stdIOLinesNext), args[0].(*lfile.File), nil}, nil
}

func stdIOFileSeek(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:seek", "file", "~string", "~number"); err != nil {
		return nil, err
	}
	file := args[0].(*lfile.File)
	whence := "cur"
	if len(args) > 1 {
		whence = args[1].(string)
	}
	offset := int64(0)
	if len(args) > 2 {
		offset = toInt(args[2])
	}

	pos, err := file.Seek(whence, offset)
	if err != nil {
		return []any{nil, err.Error()}, nil //nolint:nilerr
	}
	return []any{pos}, nil
}

func stdIOFileSetvbuf(*VM, []any) ([]any, error) {
	// not supported.
	return []any{true}, nil
}

func stdIOPOpen(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.popen", "string", "~string"); err != nil {
		return nil, err
	}
	mode := "r"
	if len(args) > 1 {
		mode = args[1].(string)
	}
	newFile, err := lfile.POpen(args[0].(string), mode)
	return []any{newFile}, err
}
