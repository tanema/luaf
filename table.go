package luaf

import (
	"bytes"
	"fmt"
)

type (
	Table struct {
		val       []Value
		hashtable map[any]Value
		metatable *Table
		keyCache  []any
	}
	metaMethod string
)

const (
	metaAdd      metaMethod = "__add"
	metaSub      metaMethod = "__sub"
	metaMul      metaMethod = "__mul"
	metaDiv      metaMethod = "__div"
	metaMod      metaMethod = "__mod"
	metaPow      metaMethod = "__pow"
	metaUNM      metaMethod = "__unm"
	metaIDiv     metaMethod = "__idiv"
	metaBAnd     metaMethod = "__band"
	metaBOr      metaMethod = "__bor"
	metaBXOr     metaMethod = "__bxor"
	metaBNot     metaMethod = "__bnot"
	metaShl      metaMethod = "__shl"
	metaShr      metaMethod = "__shr"
	metaConcat   metaMethod = "__concat"
	metaLen      metaMethod = "__len"
	metaEq       metaMethod = "__eq"
	metaLt       metaMethod = "__lt"
	metaLe       metaMethod = "__le"
	metaIndex    metaMethod = "__index"
	metaNewIndex metaMethod = "__newindex"
	metaCall     metaMethod = "__call"
	metaClose    metaMethod = "__close"
	// TODO Not implemented yet
	metaGC        metaMethod = "__gc"        // finalizer good for closing connections or files
	metaToString  metaMethod = "__tostring"  // allow custom to string behaviour
	metaPairs     metaMethod = "__pairs"     // allow custom pairs behaviour
	metaName      metaMethod = "__name"      // fallback if __string is not defined
	metaMode      metaMethod = "__mode"      // might not use, used for weak reference gc which we don't do
	metaMetaTable metaMethod = "__metatable" // allow custom getmetatable
)

func NewTable(arr []Value, hash map[any]Value) *Table {
	if hash == nil {
		hash = map[any]Value{}
	}
	keycache := []any{}
	for key := range hash {
		keycache = append(keycache, key)
	}
	return &Table{
		val:       arr,
		hashtable: hash,
		keyCache:  keycache,
	}
}

func NewSizedTable(arraySize, tableSize int) *Table {
	return &Table{
		val:       make([]Value, 0, arraySize),
		hashtable: make(map[any]Value, tableSize),
	}
}
func (t *Table) Type() string { return "table" }
func (t *Table) Val() any     { return nil }
func (t *Table) Keys() []any  { return t.keyCache }
func (t *Table) Meta() *Table { return t.metatable }
func (t *Table) String() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "{")
	for _, v := range t.val {
		if v != nil {
			fmt.Fprintf(&buf, " %s", v)
		}
	}
	for _, key := range t.Keys() {
		val := t.hashtable[key]
		fmt.Fprintf(&buf, " %s = %s", key, val)
	}
	fmt.Fprint(&buf, " }")
	return buf.String()
}

func (t *Table) Index(key Value) (Value, error) {
	switch keyval := key.(type) {
	case *Integer:
		if i := keyval.val; i > 0 && int(i) <= len(t.val) {
			return t.val[i-1], nil
		} else if int(i) > len(t.val) {
			return &Nil{}, nil
		}
	case *Nil:
		return nil, fmt.Errorf("table index is nil")
	}
	val, ok := t.hashtable[toKey(key)]
	if !ok {
		return &Nil{}, nil
	}
	return val, nil
}

func (t *Table) SetIndex(key, val Value) error {
	switch keyval := key.(type) {
	case *Integer:
		if i := keyval.val; i >= 0 {
			if int(i) > len(t.val) {
				t.val = t.val[:cap(t.val)]
			}
			t.val[i] = val
			return nil
		}
	case *Nil:
		return fmt.Errorf("table index is nil")
	}
	fmtKey := toKey(key)
	_, exists := t.hashtable[fmtKey]
	if !exists {
		t.keyCache = append(t.keyCache, fmtKey)
	}
	if _, isNil := val.(*Nil); isNil {
		for i, kc := range t.keyCache {
			if fmtKey == kc {
				t.keyCache = t.keyCache[:i+copy(t.keyCache[i:], t.keyCache[i+1:])]
				break
			}
		}
		delete(t.hashtable, fmtKey)
	} else {
		t.hashtable[fmtKey] = val
	}
	return nil
}
