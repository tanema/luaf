package runtime

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tanema/luaf/internal/conf"
	"github.com/tanema/luaf/internal/i18n"
	"github.com/tanema/luaf/internal/parse"
)

type decodeStructTest struct {
	Name     string `lua:"name"`
	Count    int
	Tags     []string          `lua:"tags"`
	Ptr      *int              `lua:"ptr"`
	Nested   *decodeStructTest `lua:"nested"`
	Any      any               `lua:"any"`
	unexport bool
}

func TestFromLuaValue_Simple(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		var out string
		require.NoError(t, FromLuaValue("hi", &out))
		assert.Equal(t, "hi", out)
	})
	t.Run("bytes", func(t *testing.T) {
		t.Parallel()
		var out []byte
		require.NoError(t, FromLuaValue("hi", &out))
		assert.Equal(t, []byte("hi"), out)
	})
	t.Run("runes", func(t *testing.T) {
		t.Parallel()
		var out []rune
		require.NoError(t, FromLuaValue("hi", &out))
		assert.Equal(t, []rune("hi"), out)
	})
	t.Run("named string", func(t *testing.T) {
		t.Parallel()
		var out i18n.LocaleCategory
		require.NoError(t, FromLuaValue("all", &out))
		assert.Equal(t, i18n.CategoryALL, out)
	})
	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		var out bool
		require.NoError(t, FromLuaValue(true, &out))
		assert.True(t, out)
	})
	t.Run("int", func(t *testing.T) {
		t.Parallel()
		var out int
		require.NoError(t, FromLuaValue(int64(42), &out))
		assert.Equal(t, 42, out)
	})
	t.Run("uint8", func(t *testing.T) {
		t.Parallel()
		var out uint8
		require.NoError(t, FromLuaValue(int64(255), &out))
		assert.Equal(t, uint8(255), out)
	})
	t.Run("named int", func(t *testing.T) {
		t.Parallel()
		var out namedInt
		require.NoError(t, FromLuaValue(int64(3), &out))
		assert.Equal(t, namedInt(3), out)
	})
	t.Run("int into float", func(t *testing.T) {
		t.Parallel()
		var out float32
		require.NoError(t, FromLuaValue(int64(3), &out))
		assert.InDelta(t, float32(3), out, 0)
	})
	t.Run("float", func(t *testing.T) {
		t.Parallel()
		var out float64
		require.NoError(t, FromLuaValue(1.5, &out))
		assert.InDelta(t, 1.5, out, 0)
	})
	t.Run("integral float into int", func(t *testing.T) {
		t.Parallel()
		var out int
		require.NoError(t, FromLuaValue(255.0, &out))
		assert.Equal(t, 255, out)
	})
	t.Run("pointer allocates", func(t *testing.T) {
		t.Parallel()
		var out *int
		require.NoError(t, FromLuaValue(int64(7), &out))
		require.NotNil(t, out)
		assert.Equal(t, 7, *out)
	})
	t.Run("pointer does not mutate shared value", func(t *testing.T) {
		t.Parallel()
		shared := 1
		out := &shared
		require.NoError(t, FromLuaValue(int64(7), &out))
		assert.Equal(t, 7, *out)
		assert.Equal(t, 1, shared)
	})
	t.Run("nil leaves value unchanged", func(t *testing.T) {
		t.Parallel()
		out := 5
		require.NoError(t, FromLuaValue(nil, &out))
		assert.Equal(t, 5, out)
	})
	t.Run("lua values passthrough", func(t *testing.T) {
		t.Parallel()
		tbl := NewTable(nil, nil)
		var out *Table
		require.NoError(t, FromLuaValue(tbl, &out))
		assert.Same(t, tbl, out)

		fn := Fn("test", nil)
		var fnOut *GoFunc
		require.NoError(t, FromLuaValue(fn, &fnOut))
		assert.Same(t, fn, fnOut)
	})
	t.Run("file", func(t *testing.T) {
		t.Parallel()
		var out *os.File
		require.NoError(t, FromLuaValue(ToLuaFile(os.Stdout), &out))
		assert.Same(t, os.Stdout, out)
	})
	t.Run("text unmarshaler", func(t *testing.T) {
		t.Parallel()
		var out *i18n.Locale
		require.NoError(t, FromLuaValue("en", &out))
		require.NotNil(t, out)
		assert.Equal(t, "en", out.String())
	})
}

func TestFromLuaValue_Errors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		val  any
		dst  any
	}{
		{name: "non pointer", val: "hi", dst: ""},
		{name: "nil pointer", val: "hi", dst: (*string)(nil)},
		{name: "string into int", val: "hi", dst: new(int)},
		{name: "bool into string", val: true, dst: new(string)},
		{name: "int overflow", val: int64(256), dst: new(uint8)},
		{name: "negative into uint", val: int64(-1), dst: new(uint)},
		{name: "fractional float into int", val: 1.5, dst: new(int)},
		{name: "float overflows float32", val: math.MaxFloat64, dst: new(float32)},
		{name: "table into int", val: NewTable(nil, nil), dst: new(int)},
		{name: "table too long for array", val: NewTable([]any{int64(1), int64(2)}, nil), dst: new([1]int)},
		{name: "bad struct field", val: NewTable(nil, map[any]any{"name": int64(1)}), dst: new(decodeStructTest)},
		{name: "bad slice element", val: NewTable([]any{"a"}, nil), dst: new([]int)},
		{name: "bad map key", val: NewTable(nil, map[any]any{"a": int64(1)}), dst: new(map[int]int)},
		{name: "unmarshaler with non string", val: int64(1), dst: new(*i18n.Locale)},
		{name: "unmarshaler with bad text", val: "not a locale", dst: new(*i18n.Locale)},
		{name: "non os file into os file", val: &File{Path: "x", handle: &wcfile{}}, dst: new(*os.File)},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, FromLuaValue(tc.val, tc.dst))
		})
	}
}

func TestFromLuaValue_Table(t *testing.T) {
	t.Parallel()

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		var out []int
		require.NoError(t, FromLuaValue(NewTable([]any{int64(1), 2.0, int64(3)}, nil), &out))
		assert.Equal(t, []int{1, 2, 3}, out)
	})
	t.Run("array zero fills", func(t *testing.T) {
		t.Parallel()
		out := [3]string{"x", "y", "z"}
		require.NoError(t, FromLuaValue(NewTable([]any{"a"}, nil), &out))
		assert.Equal(t, [3]string{"a", "", ""}, out)
	})
	t.Run("map", func(t *testing.T) {
		t.Parallel()
		var out map[string]int
		require.NoError(t, FromLuaValue(NewTable(nil, map[any]any{"a": int64(1), "b": int64(2)}), &out))
		assert.Equal(t, map[string]int{"a": 1, "b": 2}, out)
	})
	t.Run("map from array part", func(t *testing.T) {
		t.Parallel()
		var out map[int]string
		require.NoError(t, FromLuaValue(NewTable([]any{"a", nil, "c"}, nil), &out))
		assert.Equal(t, map[int]string{1: "a", 3: "c"}, out)
	})
	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		out := decodeStructTest{Count: 9, unexport: true}
		tbl := NewTable(nil, map[any]any{
			"name":   "test",
			"tags":   NewTable([]any{"x", "y"}, nil),
			"ptr":    int64(7),
			"nested": NewTable(nil, map[any]any{"name": "inner"}),
			"any":    NewTable([]any{int64(1)}, nil),
			"Count":  int64(3),
		})
		require.NoError(t, FromLuaValue(tbl, &out))
		assert.Equal(t, "test", out.Name)
		assert.Equal(t, 3, out.Count)
		assert.Equal(t, []string{"x", "y"}, out.Tags)
		require.NotNil(t, out.Ptr)
		assert.Equal(t, 7, *out.Ptr)
		require.NotNil(t, out.Nested)
		assert.Equal(t, "inner", out.Nested.Name)
		assert.Equal(t, []any{int64(1)}, out.Any)
		assert.True(t, out.unexport)
	})
	t.Run("struct keeps fields missing from table", func(t *testing.T) {
		t.Parallel()
		out := decodeStructTest{Name: "keep", Count: 9}
		require.NoError(t, FromLuaValue(NewTable(nil, map[any]any{"Count": int64(1)}), &out))
		assert.Equal(t, "keep", out.Name)
		assert.Equal(t, 1, out.Count)
	})
	t.Run("any with sequence", func(t *testing.T) {
		t.Parallel()
		var out any
		require.NoError(t, FromLuaValue(NewTable([]any{"a", NewTable([]any{int64(1)}, nil)}, nil), &out))
		assert.Equal(t, []any{"a", []any{int64(1)}}, out)
	})
	t.Run("any with mixed table", func(t *testing.T) {
		t.Parallel()
		var out any
		require.NoError(t, FromLuaValue(NewTable([]any{"a", nil}, map[any]any{"k": true}), &out))
		assert.Equal(t, map[any]any{int64(1): "a", "k": true}, out)
	})
	t.Run("any with scalar", func(t *testing.T) {
		t.Parallel()
		var out any
		require.NoError(t, FromLuaValue(int64(1), &out))
		assert.Equal(t, int64(1), out)
	})
}

type decodeNodeTest struct {
	Name string          `lua:"name"`
	Next *decodeNodeTest `lua:"next"`
}

func TestFromLuaValue_Cycles(t *testing.T) {
	t.Parallel()

	selfRef := func() *Table {
		tbl := NewTable(nil, nil)
		require.NoError(t, tbl.Set("name", "a"))
		require.NoError(t, tbl.Set("next", tbl))
		return tbl
	}

	t.Run("any with self reference", func(t *testing.T) {
		t.Parallel()
		var out any
		require.ErrorIs(t, FromLuaValue(selfRef(), &out), errCyclicTable)
	})
	t.Run("any with cycle in array part", func(t *testing.T) {
		t.Parallel()
		tbl := NewTable(nil, nil)
		require.NoError(t, tbl.Set(int64(1), tbl))
		var out any
		require.ErrorIs(t, FromLuaValue(tbl, &out), errCyclicTable)
	})
	t.Run("recursive struct", func(t *testing.T) {
		t.Parallel()
		var out decodeNodeTest
		require.ErrorIs(t, FromLuaValue(selfRef(), &out), errCyclicTable)
	})
	t.Run("indirect cycle", func(t *testing.T) {
		t.Parallel()
		a := NewTable(nil, nil)
		b := NewTable(nil, map[any]any{"name": "b", "next": a})
		require.NoError(t, a.Set("name", "a"))
		require.NoError(t, a.Set("next", b))
		var out decodeNodeTest
		require.ErrorIs(t, FromLuaValue(a, &out), errCyclicTable)
	})
	t.Run("shared table is not a cycle", func(t *testing.T) {
		t.Parallel()
		shared := NewTable([]any{int64(1)}, nil)
		var out any
		require.NoError(t, FromLuaValue(NewTable([]any{shared, shared}, nil), &out))
		assert.Equal(t, []any{[]any{int64(1)}, []any{int64(1)}}, out)
	})
	t.Run("global env", func(t *testing.T) {
		t.Parallel()
		vm, err := New(context.Background(), conf.New())
		require.NoError(t, err)
		var out any
		require.ErrorIs(t, vm.GetGlobal("_G", &out), errCyclicTable)
		require.NoError(t, vm.Close())
	})
}

func TestVM_SetGetGlobalConfig(t *testing.T) {
	t.Parallel()

	src := `
config.initial_stack_size = 510 / 2
config.max_regs = 100
config.strict = true
config.locale = "en"
`
	fn, err := parse.Parse("test", strings.NewReader(src), parse.ModeText, conf.New())
	require.NoError(t, err)

	cfg := conf.New()
	vm, err := New(context.Background(), cfg)
	require.NoError(t, err)

	require.NoError(t, vm.SetGlobal("config", cfg))
	_, err = vm.Eval(fn)
	require.NoError(t, err)
	require.NoError(t, vm.GetGlobal("config", &cfg))
	require.NoError(t, vm.Close())

	assert.Equal(t, 255, cfg.InitialStackSize)
	assert.Equal(t, uint8(100), cfg.MaxRegs)
	assert.True(t, cfg.Strict)
	assert.Equal(t, "en", cfg.Locale.String())
	assert.NotSame(t, i18n.DefaultLocale, cfg.Locale)
	assert.Equal(t, conf.MAXSTACKSIZE, cfg.MaxStackSize)
	assert.Equal(t, "en", i18n.DefaultLocale.String())
}
