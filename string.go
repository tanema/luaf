package luaf

type String struct{ val string }

func (s *String) Type() string   { return "string" }
func (s *String) Val() any       { return string(s.val) }
func (s *String) String() string { return string(s.val) }
func (s *String) Meta() *Table   { return stringMetaTable }

// for some reason lua implements arithmetic operations on strings that will work
// if the strings are convertable into numbers
var stringMetaTable = NewTable(nil, map[any]Value{
	"__add":  &Nil{},
	"__sub":  &Nil{},
	"__mul":  &Nil{},
	"__mod":  &Nil{},
	"__pow":  &Nil{},
	"__div":  &Nil{},
	"__idiv": &Nil{},
	"__unm":  &Nil{},
})
