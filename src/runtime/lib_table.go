package runtime

import (
	"errors"
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
			"count":  Fn("table.count", stdTableCount),
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
		if i := keyval; i >= 0 {
			ensureSize(&t.val, int(i))
			t.val[i] = val
			return nil
		}
	case nil:
		return errors.New("table index is nil")
	case *Table:
		panic("what")
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
	tbl := args[0].(*Table).val
	sep := ""
	i, j := int64(1), int64(len(tbl)-2)
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
	for k := i; k < j; k++ {
		strParts = append(strParts, ToString(tbl[k]))
	}
	return []any{strings.Join(strParts, sep)}, nil
}

func stdTableCount(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.count", "table"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	return []any{int64(len(tbl.Keys()))}, nil
}

func stdTableInsert(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.insert", "table", "value"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val)
	var val any
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
	return []any{}, nil
}

func stdTableMove(_ *VM, args []any) ([]any, error) {
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
		return []any{}, nil
	}
	chunk := make([]any, nelem)
	copy(chunk, tbl1.val[from-1:from+nelem-1])
	tbl1.val = slices.Delete(tbl1.val, int(from-1), int(from+nelem-1))
	ensureSize(&tbl2.val, int(to-1))
	tbl2.val = slices.Insert(tbl2.val, int(to-1), chunk...)
	return []any{}, nil
}

func stdTableRemove(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.remove", "table", "~number"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	i := len(tbl.val) - 1
	if len(args) > 1 {
		i = int(toInt(args[1]))
	}
	if i > len(tbl.val) || i < 0 {
		return []any{}, nil
	}
	tbl.val = slices.Delete(tbl.val, i-1, i)
	return []any{}, nil
}

func stdTablePack(_ *VM, args []any) ([]any, error) {
	return []any{NewTable(args, nil)}, nil
}

func stdTableSort(vm *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.sort", "table", "~function"); err != nil {
		return nil, err
	}
	var sortErr error
	tbl := args[0].(*Table)
	if len(args) > 1 {
		slices.SortFunc(tbl.val, func(l, r any) int {
			if sortErr != nil {
				return 0
			}
			res, err := vm.call(args[1], []any{l, r})
			if err != nil {
				sortErr = err
			}
			if len(res) > 0 {
				if toBool(res[0]) {
					return -1
				}
				return 1
			}
			return 0
		})
		return nil, sortErr
	}

	slices.SortFunc(tbl.val, func(l, r any) int {
		if sortErr != nil {
			return 0
		}
		i, err := compareVal(vm, parse.MetaLe, l, r)
		if err != nil {
			sortErr = err
		}
		return i
	})

	return nil, sortErr
}

func stdTableUnpack(_ *VM, args []any) ([]any, error) {
	if err := assertArguments(args, "table.unpack", "table"); err != nil {
		return nil, err
	}
	tbl := args[0].(*Table)
	return tbl.val, nil
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
