package shine

import "fmt"

type (
	Value interface {
		fmt.Stringer
		Type() string
		Val() any
		Bool() *Boolean
	}
	Nil     struct{}
	String  struct{ val string }
	Boolean struct{ val bool }
	Integer struct{ val int64 }
	Float   struct{ val float64 }
	//Function
	//Table
)

var (
	NilVal   = &Nil{}
	TrueVal  = &Boolean{true}
	FalseVal = &Boolean{false}
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

func (i *Integer) Type() string   { return "integer" }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }
func (i *Integer) Bool() *Boolean { return &Boolean{val: i.val != 0} }

func (f *Float) Type() string   { return "float" }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
func (f *Float) Bool() *Boolean { return &Boolean{val: f.val != 0} }
