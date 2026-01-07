package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// File is a lua file handle.
type File struct {
	Path      string
	Closed    bool
	process   *os.Process
	reader    *bufio.Reader
	handle    osFile
	isstdpipe bool
	readOnly  bool
	writeOnly bool
}

type (
	wcfile struct{ io.WriteCloser }
	osFile interface {
		io.ReadWriteCloser
		io.ReaderAt
		io.Seeker
		Stat() (os.FileInfo, error)
		Sync() error
	}
)

func ostoFile(wc io.WriteCloser) osFile             { return &wcfile{wc} }
func (w *wcfile) Read([]byte) (int, error)          { return 0, nil }
func (w *wcfile) ReadAt([]byte, int64) (int, error) { return 0, nil }
func (w *wcfile) Seek(int64, int) (int64, error)    { return 0, errors.New("cannot seek process") }
func (w *wcfile) Stat() (fs.FileInfo, error)        { return fs.FileInfo(nil), nil }
func (w *wcfile) Sync() error                       { return nil }

var (
	// Stdin is a file wrapper around stdin so that it can easily be read from.
	Stdin = &File{
		handle:    os.Stdin,
		reader:    bufio.NewReader(os.Stdin),
		Path:      "<stdin>",
		readOnly:  true,
		isstdpipe: true,
	}
	// Stdout is a file wrapper around stdout so that it can easily be written to.
	Stdout = &File{
		handle:    os.Stdout,
		Path:      "<stdout>",
		writeOnly: true,
		isstdpipe: true,
	}
	// Stderr is a file wrapper around stderr to easily write to.
	Stderr = &File{
		handle:    os.Stderr,
		Path:      "<stderr>",
		writeOnly: true,
		isstdpipe: true,
	}
)

// PopenCommand creates a platform independent exec.Cmd.
func PopenCommand(arg string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("C:\\Windows\\system32\\cmd.exe", append([]string{"/c"}, arg)...)
	}
	return exec.Command("/bin/sh", append([]string{"-c"}, arg)...)
}

// POpen will create a new command and executes it with a filewrapper around it,
// which makes it easy to read and write from.
func POpen(cmdSrc, mode string) (*File, error) {
	cmd := PopenCommand(cmdSrc)
	newFile := &File{Path: cmdSrc}
	switch mode {
	case "r":
		stderr, _ := cmd.StderrPipe()
		stdout, _ := cmd.StdoutPipe()
		newFile.reader = bufio.NewReader(io.MultiReader(stdout, stderr))
		newFile.readOnly = true
	case "w":
		stdin, _ := cmd.StdinPipe()
		newFile.handle = ostoFile(stdin)
		newFile.writeOnly = true
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	newFile.process = cmd.Process
	return newFile, nil
}

// OpenFile will create a new lua file handle with read and write permissions.
func OpenFile(path string, mode int, readOnly, writeOnly bool) (*File, error) {
	file, err := os.OpenFile(path, mode, 0o600)
	if err != nil {
		return nil, err
	}
	return &File{
		handle:    file,
		Path:      path,
		reader:    bufio.NewReader(file),
		writeOnly: writeOnly,
		readOnly:  readOnly,
	}, nil
}

// CreateTmpFile will create a temporary file.
func CreateTmpFile() (*File, error) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	return &File{
		handle: file,
		Path:   file.Name(),
		reader: bufio.NewReader(file),
	}, nil
}

func (f *File) String() string {
	return fmt.Sprintf("file %s %p", f.Path, f)
}

// Close will close and flush the file.
func (f *File) Close() error {
	if f.Closed {
		return nil
	} else if f.process != nil {
		defer func() { f.process = nil }()
		return f.process.Kill()
	} else if f.isstdpipe {
		return nil
	}
	f.Closed = true
	if err := f.handle.Sync(); err != nil {
		return err
	}
	return f.handle.Close()
}

func (f *File) Write(data string) error {
	if f.Closed {
		return errors.New("file closed")
	} else if f.readOnly {
		return errors.New("file readonly")
	}
	_, err := fmt.Fprint(f.handle, data)
	return err
}

// Sync will flush any writes to the file handle.
func (f *File) Sync() error {
	return f.handle.Sync()
}

// Seek will seek in the file from a point. From can be one of the following values:
// "set", from the beginning, "cur" the current position, "end" the end of the file.
// Offset is the offset in bytes.
func (f *File) Seek(from string, offset int64) (int64, error) {
	if f.Closed {
		return 0, errors.New("file closed")
	} else if f.process != nil {
		return 0, errors.New("cannot seek process")
	}
	whence := 1
	switch from {
	case "set":
		whence = 0
	case "cur":
		whence = 1
	case "end":
		whence = 2
	}
	return f.handle.Seek(offset, whence)
}

func (f *File) Read(formats []any) ([]any, error) {
	if f.Closed {
		return nil, errors.New("file closed")
	} else if f.writeOnly {
		return nil, errors.New("file writeonly")
	}

	results := []any{}
	for _, mode := range formats {
		switch fmode := mode.(type) {
		case int64:
			if fmode == 0 {
				_, err := f.reader.ReadByte()
				if errors.Is(err, io.EOF) {
					return results, nil
				} else if err := f.reader.UnreadByte(); err != nil {
					return nil, err
				}
				results = append(results, "")
				continue
			}
			buf := make([]byte, fmode)
			_, err := io.ReadFull(f.reader, buf)
			if errors.Is(err, io.EOF) {
				return results, nil
			} else if err != nil {
				return nil, err
			}
			results = append(results, string(buf))
		case string:
			switch fmode {
			case "n":
				var v float64
				_, err := fmt.Fscanf(f.reader, "%f", &v)
				if errors.Is(err, io.EOF) {
					return results, nil
				} else if err != nil {
					return nil, err
				}
				results = append(results, float64(v))
			case "a":
				buf, err := io.ReadAll(f.handle)
				if errors.Is(err, io.EOF) {
					return results, nil
				} else if err != nil {
					return nil, err
				}
				results = append(results, string(buf))
			case "l", "L":
				text, err := f.reader.ReadString('\n')
				if err != nil {
					return nil, err
				} else if fmode == "L" {
					results = append(results, text)
				} else {
					results = append(results, strings.TrimRight(text, "\r\n"))
				}
			default:
				return nil, fmt.Errorf("unknown read mode %v", fmode)
			}
		default:
			return nil, fmt.Errorf("unknown read mode %v", mode)
		}
	}
	return results, nil
}
