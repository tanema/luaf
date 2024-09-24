package shine

import "fmt"

type (
	Value interface {
		fmt.Stringer
		Type() string
		Val() any
		Bool() *Boolean
	}
	Nil      struct{}
	String   struct{ val string }
	Boolean  struct{ val bool }
	Integer  struct{ val int64 }
	Float    struct{ val float64 }
	Function struct{}
	Closure  struct{}
	Table    struct {
		val   []Value
		table map[Value]int
	}
)

func (n *Nil) Type() string   { return "nil" }
func (n *Nil) Val() any       { return nil }
func (n *Nil) String() string { return "nil" }
func (n *Nil) Bool() *Boolean { return &Boolean{val: false} }

func (s *String) Type() string   { return "string" }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }
func (s *String) Bool() *Boolean { return &Boolean{val: true} }

func (b *Boolean) Type() string   { return "boolean" }
func (b *Boolean) Val() any       { return bool(b.val) }
func (b *Boolean) String() string { return fmt.Sprintf("%v", b.val) }
func (b *Boolean) Bool() *Boolean { return b }
func (b *Boolean) Not() *Boolean  { return &Boolean{val: !b.val} }

func (i *Integer) Type() string   { return "number" }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }
func (i *Integer) Bool() *Boolean { return &Boolean{val: i.val != 0} }

func (f *Float) Type() string   { return "number" }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
func (f *Float) Bool() *Boolean { return &Boolean{val: f.val != 0} }

func (f *Function) Type() string   { return "function" }
func (f *Function) Val() any       { return nil }
func (f *Function) String() string { return fmt.Sprintf("function") }
func (f *Function) Bool() *Boolean { return &Boolean{val: true} }

func (c *Closure) Type() string   { return "function" }
func (c *Closure) Val() any       { return nil }
func (c *Closure) String() string { return fmt.Sprintf("function") }
func (c *Closure) Bool() *Boolean { return &Boolean{val: true} }

func NewTable() *Table {
	return &Table{
		val:   []Value{},
		table: map[Value]int{},
	}
}
func (t *Table) Type() string   { return "table" }
func (t *Table) Val() any       { return nil }
func (t *Table) String() string { return fmt.Sprintf("table{}") }
func (t *Table) Bool() *Boolean { return &Boolean{val: true} }

func findValue(all []Value, item Value) int {
	for i, v := range all {
		if v.Val() == item.Val() {
			return i
		}
	}
	return -1
}
