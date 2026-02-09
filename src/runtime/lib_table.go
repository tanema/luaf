package runtime

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/tanema/luaf/src/parse"
)

type (
	// Table is a container object in lua that acts both as an array and a map
	// It is used duing runtime but cal also be changed in go code.
	Table struct {
		val       []any
		hashtable map[any]any
		metatable *Table
		keyCache  []any
	}
)

func createTableLib() *Table {
	return &Table{
		hashtable: map[any]any{
			"concat": Fn("table.concat", stdTableConcat),
			"keys":   Fn("table.keys", stdTableKeys),
			"insert": Fn("table.insert", stdTableInsert),
			"move":   Fn("table.move", stdTableMove),
			"pack":   Fn("table.pack", stdTablePack),
			"remove": Fn("table.remove", stdTableRemove),
			"sort":   Fn("table.sort", stdTableSort),
			"unpack": Fn("table.unpack", stdTableUnpack),
		},
	}
}

// NewTable will create a new table with default values contained in it. Since
// lua tables act as both array and map, both can be passed in to set the values.
func NewTable(arr []any, hash map[any]any) *Table {
	if hash == nil {
		hash = map[any]any{}
	}
	keycache := make([]any, 0, len(hash))
	for key := range hash {
		keycache = append(keycache, key)
	}
	return &Table{
		val:       arr,
		hashtable: hash,
		keyCache:  keycache,
	}
}

func (t *Table) String() string {
	return fmt.Sprintf("table: %p", t.val)
}

func newSizedTable(arraySize, tableSize int) *Table {
	return &Table{
		val:       make([]any, 0, arraySize),
		hashtable: make(map[any]any, tableSize),
	}
}

// Keys returns the map keys for the map storage used for pairs iteration.
func (t *Table) Keys() []any { return t.keyCache }

// Get will return the value for the key. If it is an int it will get it from the
// array store, otherwise the map. Nil keys are not allowed.
func (t *Table) Get(key any) (any, error) {
	switch keyval := key.(type) {
	case int64:
		if i := keyval; i > 0 && int(i) <= len(t.val) {
			return t.val[i-1], nil
		} else if int(i) > len(t.val) {
			return nil, nil
		}
	case nil:
		return nil, errors.New("table index is nil")
	}
	val, ok := t.hashtable[toKey(key)]
	if !ok {
		return nil, nil
	}
	return val, nil
}

// Set will set a value at a given key. If the key is an int64, it will place it
// in array-like storage. Otherwise it will be put in a map. Nil keys are not
// allowed.
func (t *Table) Set(key, val any) error {
	switch keyval := key.(type) {
	case int64:
		if i := keyval - 1; i >= 0 {
			ensureSize(&t.val, int(i))
			t.val[i] = val
			return nil
		}
	case nil:
		return errors.New("table index is nil")
	}
	fmtKey := toKey(key)
	_, exists := t.hashtable[fmtKey]
	if !exists {
		t.keyCache = append(t.keyCache, fmtKey)
	}
	if val == nil {
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

func stdTableConcat(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "concat", "table", "~string", "~number", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	sep := ""
	i, j := int64(1), int64(len(tbl.val))
	if len(args) > 1 {
		sep = args[1].(string)
	}
	if len(args) > 2 {
		i = toInt(args[2])
	}
	if len(args) > 3 {
		j = toInt(args[3])
	}

	strParts := []string{}
	for k := i; k <= j; k++ {
		val, err := tbl.Get(k)
		if err != nil {
			return nil, err
		} else if val == nil {
			return nil, fmt.Errorf("invalid value (nil) at index %v in table for 'concat'", k)
		}
		strParts = append(strParts, ToString(val))
	}
	return []any{strings.Join(strParts, sep)}, nil
}

func stdTableKeys(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.keys", "table"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	return []any{NewTable(tbl.Keys(), nil)}, nil
}

func stdTableInsert(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.insert", "table", "value"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	if len(args) < 3 {
		tbl.val = append(tbl.val, args[1])
		return []any{}, nil
	} else if !isNumber(args[1]) {
		return nil, argumentErr(2, "table.insert", errors.New("number expected, got string"))
	}

	i := int(toInt(args[1]))
	val := args[2]
	if i <= 0 || i > len(tbl.val)+1 {
		return nil, argumentErr(2, "table.insert", errors.New("position out of bounds"))
	}

	if i == len(tbl.val)+1 {
		tbl.val = append(tbl.val, val)
		return []any{}, nil
	}

	tbl.val = slices.Insert(tbl.val, i-1, val)
	return []any{}, nil
}

func stdTableMove(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.move", "table", "number", "number", "number", "~table"); err != nil {
		return nil, err
	}
	tbl1 := args[0].(*Table)
	tbl2 := tbl1
	from := max(toInt(args[1])-1, 0)
	end := min(toInt(args[2])-1, int64(len(tbl1.val)))
	to := max(toInt(args[3])-1, 0)
	if len(args) > 4 {
		tbl2 = args[4].(*Table)
	}
	count := end - from + 1

	if end < from {
		return []any{tbl2}, nil
	} else if to > math.MaxInt64-count+1 {
		return nil, argumentErr(4, "table.move", errors.New("destination wrap around"))
	}

	if to >= int64(len(tbl2.val)) {
		ensureSize(&tbl2.val, int(to+count))
	}

	tbl2.val = slices.Insert(tbl2.val, int(to), tbl1.val[from:end]...)
	return []any{tbl2}, nil
}

func stdTableRemove(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.remove", "table", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val) - 1
	if len(args) == 1 && len(tbl.val) == 0 {
		return []any{tbl}, nil
	} else if len(args) > 1 {
		i = int(toInt(args[1])) - 1
		if i > len(tbl.val)-1 || i < 0 {
			return nil, argumentErr(2, "table.remove", errors.New("position out of bounds"))
		}
	}
	value := tbl.val[i]
	tbl.val = slices.Delete(tbl.val, i, i)
	return []any{value}, nil
}

func stdTablePack(_ *VM, args []any) ([]any, error) {
	return []any{NewTable(args, nil)}, nil
}

func stdTableSort(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.sort", "table", "~function"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	if len(args) > 1 {
		return []any{}, sortTableFunc(vm, args[1], tbl.val)
	}
	return []any{}, sortTableDefault(vm, tbl.val)
}

func sortTableFunc(vm *VM, fn any, tbl []any) error {
	var err error
	slices.SortFunc(tbl, func(l, r any) int {
		if err != nil {
			return 0
		}
		var res []any
		res, err = vm.call(fn, []any{l, r})
		if err == nil && len(res) > 0 {
			if toBool(res[0]) {
				return -1
			}
			return 1
		}
		return 0
	})
	return err
}

func sortTableDefault(vm *VM, tbl []any) error {
	var err error
	slices.SortFunc(tbl, func(l, r any) int {
		if err == nil {
			var i int
			i, err = compareVal(vm, parse.MetaLe, l, r)
			return i
		}
		return 0
	})
	return err
}

func stdTableUnpack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.unpack", "table", "~number", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := 0
	j := len(tbl.val)
	if len(args) > 1 {
		i = max(int(toInt(args[1]))-1, 0)
	}
	if i >= len(tbl.val) {
		return []any{nil}, nil
	}
	if len(args) > 2 {
		j = min(int(toInt(args[2])), len(tbl.val))
	}
	if j <= i {
		return []any{nil}, nil
	}
	return tbl.val[i:j], nil
}

func argsToTableValues(clargs []string) ([]any, map[any]any) {
	splitidx := slices.Index(clargs, "--")
	if splitidx == -1 {
		splitidx = len(clargs)
	} else {
		splitidx++
	}

	argValues := make([]any, len(clargs))
	for i, a := range clargs {
		argValues[i] = a
	}

	tbl := map[any]any{}
	for i := range splitidx {
		tbl[int64(-(splitidx-i)+1)] = argValues[i]
	}

	return argValues[splitidx:], tbl
}
