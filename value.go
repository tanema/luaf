package shine

import "fmt"

type (
	Value interface {
		Type() string
		Val() any
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

func (s *String) Type() string   { return "string" }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }

func (b *Boolean) Type() string   { return "boolean" }
func (b *Boolean) Val() any       { return bool(b.val) }
func (b *Boolean) String() string { return fmt.Sprintf("%v", b.val) }

func (i *Integer) Type() string   { return "integer" }
func (i *Integer) Val() any       { return int64(i.val) }
func (i *Integer) String() string { return fmt.Sprintf("%v", i.val) }

func (f *Float) Type() string   { return "float" }
func (f *Float) Val() any       { return float64(f.val) }
func (f *Float) String() string { return fmt.Sprintf("%v", f.val) }
