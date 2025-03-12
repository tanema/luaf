package luaf

import (
	"fmt"
	"slices"
	"strings"
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
	metaToString metaMethod = "__tostring"
	metaName     metaMethod = "__name"
	metaPairs    metaMethod = "__pairs"
	metaMeta     metaMethod = "__metatable"
	metaGC       metaMethod = "__gc"
)

func createTableLib() *Table {
	return &Table{
		hashtable: map[any]Value{
			"concat": Fn("table.concat", stdTableConcat),
			"insert": Fn("table.insert", stdTableInsert),
			"move":   Fn("table.move", stdTableMove),
			"pack":   Fn("table.pack", stdTablePack),
			"remove": Fn("table.remove", stdTableRemove),
			"sort":   Fn("table.sort", stdTableSort),
			"unpack": Fn("table.unpack", stdTableUnpack),
		},
	}
}

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
func (t *Table) Type() string   { return "table" }
func (t *Table) Val() any       { return nil }
func (t *Table) Keys() []any    { return t.keyCache }
func (t *Table) Meta() *Table   { return t.metatable }
func (t *Table) String() string { return fmt.Sprintf("table %p", t) }

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
			ensureSize(&t.val, int(i))
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

func stdTableConcat(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "concat", "table", "~string", "~number", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table).val
	sep := ""
	i, j := int64(1), int64(len(tbl)-2)
	if len(args) > 1 {
		sep = args[1].(*String).val
	}
	if len(args) > 2 {
		i = toInt(args[2])
	}
	if len(args) > 3 {
		j = toInt(args[3])
	}
	strParts := []string{}
	for k := i; k < j; k++ {
		strParts = append(strParts, tbl[k].String())
	}
	return []Value{&String{val: strings.Join(strParts, sep)}}, nil
}

func stdTableInsert(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "table.insert", "table", "value"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val)
	var val Value
	if isNumber(args[1]) && len(args) > 2 {
		i = int(toInt(args[1]))
		val = args[2]
	} else {
		val = args[1]
	}
	if i <= 0 {
		i = 1
	}
	ensureSize(&tbl.val, i-1)
	tbl.val = slices.Insert(tbl.val, i-1, val)
	return []Value{}, nil
}

func stdTableMove(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "table.move", "table", "number", "number", "number", "~table"); err != nil {
		return nil, err
	}
	tbl1 := args[0].(*Table)
	tbl2 := tbl1
	from := toInt(args[1])
	nelem := toInt(args[2])
	to := toInt(args[3])
	if len(args) > 4 {
		tbl2 = args[4].(*Table)
	}
	if int(from) > len(tbl1.val) || from < 0 {
		return []Value{}, nil
	}
	chunk := make([]Value, nelem)
	copy(chunk, tbl1.val[from-1:from+nelem-1])
	tbl1.val = slices.Delete(tbl1.val, int(from-1), int(from+nelem-1))
	ensureSize(&tbl2.val, int(to-1))
	tbl2.val = slices.Insert(tbl2.val, int(to-1), chunk...)
	return []Value{}, nil
}

func stdTableRemove(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "table.remove", "table", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val) - 1
	if len(args) > 1 {
		i = int(toInt(args[1]))
	}
	if i > len(tbl.val) || i < 0 {
		return []Value{}, nil
	}
	tbl.val = slices.Delete(tbl.val, i-1, i)
	return []Value{}, nil
}

func stdTablePack(vm *VM, args []Value) ([]Value, error) {
	return []Value{NewTable(args, nil)}, nil
}

func stdTableSort(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "table.sort", "table", "~function"); err != nil {
		return nil, err
	}
	var sortErr error
	tbl := args[0].(*Table)
	if len(args) > 1 {
		slices.SortFunc(tbl.val, func(l, r Value) int {
			if sortErr != nil {
				return 0
			}
			res, err := vm.call(args[1], []Value{l, r})
			if err != nil {
				sortErr = err
			}
			if len(res) > 0 {
				if toBool(res[0]).val {
					return -1
				}
				return 1
			}
			return 0
		})
		return nil, sortErr
	}

	slices.SortFunc(tbl.val, func(l, r Value) int {
		if sortErr != nil {
			return 0
		}
		i, err := compareVal(vm, metaLe, l, r)
		if err != nil {
			sortErr = err
		}
		return i
	})

	return nil, sortErr
}

func stdTableUnpack(vm *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "table.unpack", "table"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	return tbl.val, nil
}
