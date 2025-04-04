package luaf

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type (
	osFile interface {
		io.ReadWriteCloser
		io.ReaderAt
		io.Seeker
		Stat() (os.FileInfo, error)
		Sync() error
	}
	File struct {
		process   *os.Process
		reader    *bufio.Reader
		handle    osFile
		path      string
		isstdpipe bool
		closed    bool
		readOnly  bool
		writeOnly bool
	}
)

func NewFile(path string, mode int, readOnly, writeOnly bool) (*File, error) {
	file, err := os.OpenFile(path, mode, 0o600)
	if err != nil {
		return nil, err
	}
	return &File{
		handle:    file,
		path:      path,
		reader:    bufio.NewReader(file),
		writeOnly: writeOnly,
		readOnly:  readOnly,
	}, nil
}

func (f *File) Close() error {
	if f.closed {
		return nil
	} else if f.process != nil {
		defer func() { f.process = nil }()
		return f.process.Kill()
	} else if f.isstdpipe {
		return nil
	}
	f.closed = true
	if err := f.handle.Sync(); err != nil {
		return err
	}
	return f.handle.Close()
}

func (f *File) Write(vm *VM, args []Value) ([]Value, error) {
	if f.closed {
		return nil, argumentErr(1, "file:write", errors.New("file closed"))
	} else if f.readOnly {
		return nil, argumentErr(1, "file:write", errors.New("file readonly"))
	}
	strParts := make([]string, len(args))
	for i, arg := range args {
		str, err := toString(vm, arg)
		if err != nil {
			return nil, err
		}
		strParts[i] = str.val
	}
	_, err := fmt.Fprint(f.handle, strings.Join(strParts, ""))
	if err != nil {
		return nil, err
	}
	return []Value{f}, nil
}

func (f *File) Read(_ *VM, args []Value) ([]Value, error) {
	if f.closed {
		return nil, argumentErr(1, "file:read", errors.New("file closed"))
	} else if f.writeOnly {
		return nil, argumentErr(1, "file:read", errors.New("file writeonly"))
	}

	formats := []Value{&String{val: "l"}}
	if len(args) > 0 {
		formats = args
	}
	results := []Value{}
formats_loop:
	for _, mode := range formats {
		switch fmode := mode.(type) {
		case *Integer, *Float:
			size := toInt(fmode)
			if size == 0 {
				_, err := f.reader.ReadByte()
				if errors.Is(err, io.EOF) {
					results = append(results, &Nil{})
					break formats_loop
				} else if err := f.reader.UnreadByte(); err != nil {
					results = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
					break formats_loop
				}
				results = append(results, &String{})
				continue
			}
			buf := make([]byte, size)
			_, err := io.ReadFull(f.reader, buf)
			if errors.Is(err, io.EOF) {
				results = append(results, &Nil{})
				break formats_loop
			} else if err != nil {
				results = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
				break formats_loop
			}
			results = append(results, &String{val: string(buf)})
		case *String:
			switch fmode.val {
			case "n":
				var v float64
				_, err := fmt.Fscanf(f.reader, "%f", &v)
				if errors.Is(err, io.EOF) {
					results = append(results, &Nil{})
					break formats_loop
				} else if err != nil {
					results = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
					break formats_loop
				}
				results = append(results, &Float{val: v})
			case "a":
				buf, err := io.ReadAll(f.handle)
				if errors.Is(err, io.EOF) {
					results = append(results, &String{})
					break formats_loop
				} else if err != nil {
					results = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
					break formats_loop
				}
				results = append(results, &String{val: string(buf)})
			case "l", "L":
				text, err := f.reader.ReadString('\n')
				if err != nil {
					results = []Value{&Nil{}, &String{val: err.Error()}, &Integer{val: 1}}
					break formats_loop
				} else if fmode.val == "L" {
					results = append(results, &String{val: text})
				} else {
					results = append(results, &String{val: strings.TrimRight(text, "\r\n")})
				}
			default:
				results = []Value{&Nil{}, &String{val: "unknown read mode " + fmode.val}, &Integer{val: 1}}
				break formats_loop
			}
		default:
			results = []Value{&Nil{}, &String{val: "unknown read mode " + mode.String()}, &Integer{val: 1}}
			break formats_loop
		}
	}
	return results, nil
}

func (f *File) Type() string   { return string(typeFile) }
func (f *File) Val() any       { return f }
func (f *File) String() string { return fmt.Sprintf("file %s %p", f.path, f) }
func (f *File) Meta() *Table   { return fileMetatable }

type wcfile struct {
	io.WriteCloser
}

func (w *wcfile) Read([]byte) (int, error)          { return 0, nil }
func (w *wcfile) ReadAt([]byte, int64) (int, error) { return 0, nil }
func (w *wcfile) Seek(int64, int) (int64, error)    { return 0, nil }
func (w *wcfile) Stat() (fs.FileInfo, error)        { return fs.FileInfo(nil), nil }
func (w *wcfile) Sync() error                       { return nil }

func writerCloserToFile(wc io.WriteCloser) osFile {
	return &wcfile{wc}
}
