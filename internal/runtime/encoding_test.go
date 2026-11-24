package runtime

import (
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tanema/luaf/internal/i18n"
)

type namedInt int

type encodeStructTest struct {
	Name     string `lua:"name"`
	Count    int
	Tags     []string `lua:"tags"`
	Ptr      *int     `lua:"ptr"`
	unexport bool
}

func TestToLuaValue(t *testing.T) {
	t.Parallel()

	tbl := NewTable(nil, nil)
	gofn := Fn("test", nil)
	seven := 7

	testcases := []struct {
		name string
		in   any
		out  any
	}{
		{name: "nil", in: nil, out: nil},
		{name: "nil pointer", in: (*int)(nil), out: nil},
		{name: "nil slice", in: []int(nil), out: nil},
		{name: "bool", in: true, out: true},
		{name: "string", in: "hello", out: "hello"},
		{name: "bytes", in: []byte("hello"), out: "hello"},
		{name: "runes", in: []rune("hello"), out: "hello"},
		{name: "int", in: 42, out: int64(42)},
		{name: "int8", in: int8(-42), out: int64(-42)},
		{name: "uint16", in: uint16(42), out: int64(42)},
		{name: "max uint64 that fits", in: uint64(math.MaxInt64), out: int64(math.MaxInt64)},
		{name: "named int", in: namedInt(3), out: int64(3)},
		{name: "named string", in: i18n.CategoryALL, out: "all"},
		{name: "float32", in: float32(1.5), out: float64(1.5)},
		{name: "float64", in: 1.5, out: 1.5},
		{name: "pointer", in: &seven, out: int64(7)},
		{name: "table passthrough", in: tbl, out: tbl},
		{name: "gofunc passthrough", in: gofn, out: gofn},
		{name: "text marshaler", in: i18n.DefaultLocale, out: i18n.DefaultLocale.String()},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val, err := ToLuaValue(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.out, val)
		})
	}
}

func TestToLuaValue_Errors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		in   any
	}{
		{name: "uint64 overflow", in: uint64(math.MaxUint64)},
		{name: "complex", in: complex(1, 2)},
		{name: "channel", in: make(chan int)},
		{name: "wrong func signature", in: func() {}},
		{name: "nested error", in: []any{complex(1, 2)}},
		{name: "struct field error", in: struct{ C complex64 }{}},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ToLuaValue(tc.in)
			require.Error(t, err)
		})
	}
}

func TestToLuaValue_Func(t *testing.T) {
	t.Parallel()

	val, err := ToLuaValue(func(*VM, []any) ([]any, error) { return []any{int64(1)}, nil })
	require.NoError(t, err)
	fn, ok := val.(*GoFunc)
	require.True(t, ok)
	res, err := fn.val(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1)}, res)
}

func TestToLuaValue_File(t *testing.T) {
	t.Parallel()

	val, err := ToLuaValue(os.Stdout)
	require.NoError(t, err)
	file, ok := val.(*File)
	require.True(t, ok)
	assert.Equal(t, os.Stdout, file.handle)
}

func TestToLuaValue_Slice(t *testing.T) {
	t.Parallel()

	val, err := ToLuaValue([]int{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, val.(*Table).val)

	val, err = ToLuaValue([2]string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b"}, val.(*Table).val)
}

func TestToLuaValue_Map(t *testing.T) {
	t.Parallel()

	val, err := ToLuaValue(map[string]int{"a": 1, "b": 2})
	require.NoError(t, err)
	tbl := val.(*Table)
	assertTableGet(t, tbl, "a", int64(1))
	assertTableGet(t, tbl, "b", int64(2))
	assert.ElementsMatch(t, []any{"a", "b"}, tbl.Keys())

	val, err = ToLuaValue(map[int]string{1: "a", 2: "b"})
	require.NoError(t, err)
	tbl = val.(*Table)
	assertTableGet(t, tbl, int64(1), "a")
	assertTableGet(t, tbl, int64(2), "b")
}

func TestToLuaValue_Struct(t *testing.T) {
	t.Parallel()

	seven := 7
	for _, in := range []any{
		encodeStructTest{Name: "test", Count: 3, Tags: []string{"x"}, Ptr: &seven, unexport: true},
		&encodeStructTest{Name: "test", Count: 3, Tags: []string{"x"}, Ptr: &seven, unexport: true},
	} {
		val, err := ToLuaValue(in)
		require.NoError(t, err)
		tbl := val.(*Table)
		assertTableGet(t, tbl, "name", "test")
		assertTableGet(t, tbl, "Count", int64(3))
		assertTableGet(t, tbl, "ptr", int64(7))
		tags, err := tbl.Get("tags")
		require.NoError(t, err)
		assert.Equal(t, []any{"x"}, tags.(*Table).val)
		assert.ElementsMatch(t, []any{"name", "Count", "tags", "ptr"}, tbl.Keys())
	}

	val, err := ToLuaValue(encodeStructTest{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []any{"name", "Count"}, val.(*Table).Keys())
}

func assertTableGet(t *testing.T, tbl *Table, key, expected any) {
	t.Helper()
	val, err := tbl.Get(key)
	require.NoError(t, err)
	assert.Equal(t, expected, val)
}
