package runtime

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/tanema/luaf/internal/parse"
)

var (
	defaultInput  = Stdin
	defaultOutput = Stdout
	lineReader    *readline.Instance
)

var fileMetatable *Table

func createIOLib() *Table {
	fileMetatable = &Table{
		hashtable: map[any]any{
			string(parse.MetaName):     "FILE*",
			string(parse.MetaToString): Fn("file:__tostring", stdIOFileString),
			string(parse.MetaClose):    Fn("file:__close", stdIOFileClose),
			string(parse.MetaGC):       Fn("file:__gc", stdIOFileClose),
			string(parse.MetaIndex): &Table{
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
			"stderr":   Stderr,
			"stdin":    Stdin,
			"stdout":   Stdout,
			"input":    Fn("io.input", stdIOInput),
			"output":   Fn("io.output", stdIOOutput),
			"stat":     Fn("io.stat", stdIOStat),
			"touch":    Fn("io.touch", stdIOTouch),
			"dir":      Fn("io.dir", stdIODir), //nolint:goconst
			"walk":     Fn("io.walk", stdIOWalk),
			"open":     Fn("io.open", stdIOOpen),
			"close":    Fn("io.close", stdIOClose),
			"flush":    Fn("io.flush", stdIOFlush),
			"tmpfile":  Fn("io.tmpfile", stdIOTmpfile),
			"type":     Fn("io.type", stdIOType),
			"read":     Fn("io.read", stdIORead),
			"readline": Fn("io.readline", stdIOReadline),
			"write":    Fn("io.write", stdIOWrite),
			"lines":    Fn("io.lines", stdIOLines),
			"popen":    Fn("io.popen", stdIOPOpen),
		},
	}
}

func stdIOClose(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.close", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*File)
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
	file, _ := args[0].(*File)
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
		file, _ = args[0].(*File)
	}
	return []any{ToString(file)}, nil
}

func stdIOFlush(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.flush", "~file"); err != nil {
		return nil, err
	}
	file := defaultOutput
	if len(args) > 0 {
		file, _ = args[0].(*File)
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
	file, _ := args[0].(*File)
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("problem flushing file: %v", err.Error())
	}
	return []any{}, nil
}

func stdIOStat(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.stat", "string"); err != nil {
		return nil, err
	}
	filename := args[0].(string)
	info, err := os.Stat(filename)
	if err != nil {
		return []any{nil, toLuaError(err)}, nil
	}
	linfo := map[any]any{
		"dir":   info.IsDir(),
		"mtime": info.ModTime(),
		"mode":  info.Mode().String(),
		"bytes": info.Size(),
		"perm":  info.Mode().Perm().String(),
	}
	return []any{NewTable(nil, linfo), nil}, nil
}

func stdIOTouch(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.touch", "string", "~number", "~number"); err != nil {
		return nil, err
	}
	filename := args[0].(string)
	atime := time.Now().Local()
	mtime := time.Now().Local()
	if len(args) > 1 {
		atime = time.UnixMilli(toInt(args[1]))
	}
	if len(args) > 2 {
		mtime = time.UnixMilli(toInt(args[2]))
	}
	if err := os.Chtimes(filename, mtime, atime); err != nil {
		return []any{nil, toLuaError(fmt.Errorf("error while touching file: %w", err))}, nil
	}
	return []any{true, nil}, nil
}

func stdIODir(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.dir", "string"); err != nil {
		return nil, err
	}
	dirName := args[0].(string)
	absDirPath, err := filepath.Abs(dirName)
	if err != nil {
		return []any{nil, toLuaError(err)}, nil
	} else if info, err := os.Stat(absDirPath); err != nil {
		return []any{nil, toLuaError(fmt.Errorf("problem stating dir: %w", err))}, nil
	} else if !info.IsDir() {
		return []any{nil, toLuaError(fmt.Errorf("file at path %s is not a directory", absDirPath))}, nil
	}

	dirEntries, err := os.ReadDir(absDirPath)
	if err != nil {
		return []any{nil, toLuaError(err)}, nil
	}

	entries := make([]any, len(dirEntries))
	for i, entry := range dirEntries {
		entries[i] = NewTable(nil, map[any]any{
			"path": entry.Name(),
			"dir":  entry.IsDir(),
		})
	}

	return []any{NewTable(entries, nil)}, nil
}

func stdIOWalk(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.dir", "string", "function"); err != nil {
		return nil, err
	}
	dirName := args[0].(string)
	fn := args[1]
	absDirPath, err := filepath.Abs(dirName)
	if err != nil {
		return []any{toLuaError(err)}, nil
	} else if info, err := os.Stat(absDirPath); err != nil {
		return []any{toLuaError(fmt.Errorf("problem stating dir: %w", err))}, nil
	} else if !info.IsDir() {
		return []any{toLuaError(fmt.Errorf("file at path %s is not a directory", absDirPath))}, nil
	}

	err = filepath.WalkDir(absDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info := NewTable(nil, map[any]any{
			"name": d.Name(),
			"dir":  d.IsDir(),
		})

		_, err = vm.call(fn, []any{path, info})
		return err
	})
	if err != nil {
		return []any{toLuaError(err)}, nil
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
	file, err := OpenFile(filepath, filemode, readOnly, writeOnly)
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
	newFile, err := CreateTmpFile()
	return []any{newFile}, err
}

func stdIOType(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.input", "value"); err != nil {
		return nil, err
	}
	switch f := args[0].(type) {
	case *File:
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
	var file *File
	var err error
	switch farg := args[0].(type) {
	case *File:
		file = farg
	case string:
		file, err = OpenFile(farg, os.O_RDWR, false, false)
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
	var file *File
	var err error
	switch farg := args[0].(type) {
	case *File:
		file = farg
	case string:
		file, err = OpenFile(farg, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, false, true)
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

	toWrite := args[1:]
	strParts := make([]string, len(toWrite))
	for i, arg := range toWrite {
		str, err := vm.toString(arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str
	}
	return []any{}, args[0].(*File).Write(strings.Join(strParts, ""))
}

func stdIORead(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.read", "~file|string", "~string"); err != nil {
		return nil, err
	}
	file := defaultInput
	if len(args) > 0 {
		if f, isFile := args[0].(*File); isFile {
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
	file := args[0].(*File)
	formats := []any{"l"}
	if len(args) > 1 {
		formats = args[1:]
	}
	return file.Read(formats)
}

func stdIOLinesNext(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.lines.next", "file"); err != nil {
		return nil, err
	}
	file := args[0].(*File)
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
		file = args[0].(*File)
	}
	return []any{Fn("io.lines.next", stdIOLinesNext), file, nil}, nil
}

func stdIOFileLines(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:lines", "file"); err != nil {
		return nil, err
	}
	return []any{Fn("file:lines.next", stdIOLinesNext), args[0].(*File), nil}, nil
}

func stdIOFileSeek(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "file:seek", "file", "~string", "~number"); err != nil {
		return nil, err
	}
	file := args[0].(*File)
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
	newFile, err := POpen(args[0].(string), mode)
	return []any{newFile}, err
}

func stdIOReadline(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "io.readline", "~string"); err != nil {
		return nil, err
	}
	if lineReader == nil {
		rl, err := readline.New("")
		if err != nil {
			return nil, err
		}
		lineReader = rl
	}
	prompt := ""
	if len(args) > 0 {
		prompt = args[0].(string)
	}
	lineReader.SetPrompt(prompt)
	line, err := lineReader.Readline()
	switch {
	case errors.Is(err, readline.ErrInterrupt):
		return []any{nil, "interrupt"}, nil
	case errors.Is(err, io.EOF):
		return []any{nil, "eof"}, nil
	case err != nil:
		return []any{nil, err.Error()}, nil //nolint:nilerr
	}
	return []any{line}, nil
}
