package lfile

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

type (
	wcfile struct {
		io.WriteCloser
	}
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
