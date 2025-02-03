package luaf

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"strconv"
	"strings"
)

type (
	typeName string
	callable interface {
		Call(vm *VM, nargs int64) ([]Value, error)
	}
	GoFunc func(*VM, []Value) ([]Value, error)
	Value  interface {
		fmt.Stringer
		Type() string
		Val() any
		Meta() *Table
	}
	Nil        struct{}
	Boolean    struct{ val bool }
	Integer    struct{ val int64 }
	Float      struct{ val float64 }
	ExternFunc struct{ val GoFunc }
	Closure    struct {
		val      *FnProto
		upvalues []*UpvalueBroker
	}
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
	Error struct {
		val   Value
		addr  string
		trace string
	}
)

const (
	typeUnknown typeName = "unknown" // used for type hinting
	typeString  typeName = "string"
	typeTable   typeName = "table"
	typeFunc    typeName = "function"
	typeNumber  typeName = "number"
	typeBool    typeName = "boolean"
	typeNil     typeName = "nil"
	typeClosure typeName = "closure"
	typeError   typeName = "error"
	typeFile    typeName = "file"
)

func ToValue(in any) Value {
	switch val := unifyType(in).(type) {
	case int64:
		return &Integer{val: val}
	case float64:
		return &Float{val: val}
	case bool:
		return &Boolean{val: val}
	case string:
		return &String{val: val}
	case nil:
		return &Nil{}
	case Value:
		return val
	default:
		return nil
	}
}

func toBool(in Value) *Boolean {
	switch tin := in.(type) {
	case *Error, *String, *Closure, *ExternFunc, *Table, *Integer, *Float:
		return &Boolean{val: true}
	case *Boolean:
		return tin
	default:
		return &Boolean{val: false}
	}
}

func toKey(in Value) any {
	switch tin := in.(type) {
	case *Nil:
		panic("dont use nil as a key!")
	case *String:
		return tin.val
	case *Boolean:
		return tin.val
	case *Integer:
		return tin.val
	case *Float:
		return tin.val
	default:
		return in
	}
}

func isNumber(in Value) bool {
	switch in.(type) {
	case *Integer, *Float:
		return true
	default:
		return false
	}
}

func isNil(in Value) bool {
	switch in.(type) {
	case *Nil, nil:
		return true
	default:
		return false
	}
}

func isString(in Value) bool {
	switch in.(type) {
	case *String:
		return true
	default:
		return false
	}
}

func toInt(val Value) int64 {
	switch tval := val.(type) {
	case *Integer:
		return tval.val
	case *Float:
		return int64(tval.val)
	default:
		return int64(math.NaN())
	}
}

func toFloat(val Value) float64 {
	switch tval := val.(type) {
	case *Integer:
		return float64(tval.val)
	case *Float:
		return tval.val
	default:
		return math.NaN()
	}
}

func toNumber(in Value, base int) Value {
	switch tin := in.(type) {
	case *Integer, *Float:
		return in
	case *String:
		if strings.Contains(tin.val, ".") {
			fval, err := strconv.ParseFloat(tin.val, 64)
			if err != nil {
				return &Nil{}
			}
			return &Float{val: fval}
		}
		ival, err := strconv.ParseInt(tin.val, base, 64)
		if err != nil {
			return &Nil{}
		}
		return &Integer{val: ival}
	default:
		return &Nil{}
	}
}

func toString(vm *VM, val Value) (*String, error) {
	didDelegate, res, err := vm.delegateMetamethod(metaToString, val)
	if err != nil {
		return nil, err
	} else if !didDelegate || len(res) == 0 {
		didDelegate, res, err := vm.delegateMetamethod(metaName, val)
		if err != nil {
			return nil, err
		} else if !didDelegate || len(res) == 0 {
			return &String{val: val.String()}, nil
		}
		return &String{val: res[0].String()}, nil
	}
	return &String{val: res[0].String()}, nil
}

func toError(vm *VM, val Value, level int) (*Error, error) {
	didDelegate, res, err := vm.delegateMetamethod(metaToString, val)
	if err != nil {
		return nil, err
	} else if didDelegate && len(res) > 0 {
		val = &String{val: res[0].String()}
	}
	newError := &Error{val: val}
	if vm.callStack.Len() > 0 && level > 0 {
		ci := vm.callStack.Back()
		for i := 0; i < level && ci.Prev() != nil; i++ {
			ci = ci.Prev()
		}
		info := ci.Value.(*callInfo)
		newError.addr = fmt.Sprintf(" %v:%v: ", info.filename, info.Line)
		newError.trace = printStackTrace(vm.callStack)
	}
	return newError, nil
}

func (err *Error) Type() string { return string(typeError) }
func (err *Error) Val() any     { return err.val }
func (err *Error) String() string {
	msg := err.addr
	if str, isStr := err.val.(*String); isStr {
		msg += ": " + str.val
	} else {
		msg += fmt.Sprintf(" (error object is a %v value)", err.val.Type())
	}
	if err.trace != "" {
		msg += "\n" + err.trace
	}
	return msg
}
func (err *Error) Error() string { return err.String() }
func (err *Error) Meta() *Table  { return nil }

func (n *Nil) Type() string   { return string(typeNil) }
func (n *Nil) Val() any       { return nil }
func (n *Nil) String() string { return "nil" }
func (n *Nil) Meta() *Table   { return nil }

func (b *Boolean) Type() string   { return string(typeBool) }
func (b *Boolean) Val() any       { return bool(b.val) }
func (b *Boolean) String() string { return fmt.Sprintf("%v", b.val) }
func (b *Boolean) Not() *Boolean  { return &Boolean{val: !b.val} }
func (b *Boolean) Meta() *Table   { return nil }

func (i *Integer) Type() string   { return string(typeNumber) }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }
func (i *Integer) Meta() *Table   { return nil }

func (f *Float) Type() string   { return string(typeNumber) }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
func (f *Float) Meta() *Table   { return nil }

func (c *Closure) Type() string   { return string(typeFunc) }
func (c *Closure) Val() any       { return c.val }
func (c *Closure) String() string { return fmt.Sprintf("function %p", c) }
func (c *Closure) Meta() *Table   { return nil }
func (c *Closure) Call(vm *VM, nargs int64) ([]Value, error) {
	if diff := int64(c.val.Arity) - nargs; nargs > 0 && diff > 0 {
		for i := nargs; i <= int64(c.val.Arity); i++ {
			if err := vm.SetStack(i, &Nil{}); err != nil {
				return nil, err
			}
		}
	}
	values, _, err := vm.eval(c.val, c.upvalues)
	return values, err
}

func (f *ExternFunc) Type() string   { return string(typeFunc) }
func (f *ExternFunc) Val() any       { return f.val }
func (f *ExternFunc) String() string { return fmt.Sprintf("function %p", f) }
func (f *ExternFunc) Meta() *Table   { return nil }
func (f *ExternFunc) Call(vm *VM, nargs int64) ([]Value, error) {
	return f.val(vm, vm.argsFromStack(0, nargs))
}

func NewFile(path string, mode int, readOnly, writeOnly bool) (*File, error) {
	file, err := os.OpenFile(path, mode, 0600)
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
	defer func() {
		f.process = nil
		f.closed = !f.isstdpipe
	}()
	if f.closed {
		return nil
	} else if f.process != nil {
		return f.process.Kill()
	} else if f.isstdpipe {
		return nil
	}
	return f.handle.Close()
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
