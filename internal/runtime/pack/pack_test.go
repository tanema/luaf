package pack

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPack(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc   string
		format string
		data   []any
		want   string
	}{
		{desc: "signed byte", format: "b", data: []any{int64(-1)}, want: "\xff"},
		{desc: "unsigned byte", format: "B", data: []any{int64(0xff)}, want: "\xff"},
		{desc: "signed short little endian", format: "<h", data: []any{int64(-1)}, want: "\xff\xff"},
		{desc: "unsigned short big endian", format: ">H", data: []any{int64(0xffff)}, want: "\xff\xff"},
		{desc: "arbitrary width signed int i3", format: "i3", data: []any{int64(-1)}, want: "\xff\xff\xff"},
		{desc: "arbitrary width unsigned int little endian", format: "<I3", data: []any{int64(0xAA)}, want: "\xAA\x00\x00"},
		{desc: "arbitrary width unsigned int big endian", format: ">I3", data: []any{int64(0xAA)}, want: "\x00\x00\xAA"},
		{desc: "zero terminated string", format: "z", data: []any{"hi"}, want: "hi\x00"},
		{desc: "fixed string pads short input", format: "c5", data: []any{"hi"}, want: "hi\x00\x00\x00"},
		{desc: "length prefixed string", format: "s1", data: []any{"hi"}, want: "\x02hi"},
		{desc: "padding byte", format: "x", data: nil, want: "\x00"},
		{
			desc:   "mixed endianness in one format",
			format: ">i2 <i2",
			data:   []any{int64(10), int64(20)},
			want:   "\x00\x0a\x14\x00",
		},
		{
			desc:   "alignment via !n pads before the aligned item",
			format: "!4 b i4",
			data:   []any{int64(1), int64(2)},
			want:   "\x01\x00\x00\x00\x02\x00\x00\x00",
		},
		{
			desc:   "X aligns without consuming a value, maxalign of 1 makes it a no-op",
			format: " b b Xd b Xb x", //nolint:dupword // repeated format letters, not prose
			data:   []any{int64(1), int64(2), int64(3)},
			want:   "\x01\x02\x03\x00",
		},
		{desc: "float accepts an int64 argument", format: "<f", data: []any{int64(1)}, want: "\x00\x00\x80\x3f"},
		{
			desc:   "double accepts an int64 argument",
			format: "<d",
			data:   []any{int64(1)},
			want:   "\x00\x00\x00\x00\x00\x00\xf0\x3f",
		},
		{desc: "int accepts a float64 argument", format: "<i2", data: []any{float64(258)}, want: "\x02\x01"},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			out, err := Pack(tc.format, tc.data...)
			require.NoError(t, err)
			assert.Equal(t, tc.want, out)
		})
	}
}

func TestPackErrors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc    string
		format  string
		data    []any
		wantErr string
	}{
		{desc: "zero sized integer", format: "i0", data: []any{int64(0)}, wantErr: "out of limits"},
		{desc: "integer size over max", format: "i17", data: []any{int64(0)}, wantErr: "out of limits"},
		{desc: "alignment size over max", format: "!17", data: []any{int64(0)}, wantErr: "out of limits"},
		{desc: "alignment not power of 2", format: "!4i3", data: []any{int64(0)}, wantErr: "not power of 2"},
		{desc: "trailing garbage after option", format: "i3r", data: []any{int64(0)}, wantErr: "invalid format option 'r'"},
		{desc: "unknown format option", format: "w", data: []any{int64(0)}, wantErr: "invalid format option 'w'"},
		{desc: "c missing size", format: "c", data: []any{""}, wantErr: "missing size"},
		{desc: "size digit run overflows", format: "c1" + repeatDigit(40), data: []any{""}, wantErr: "invalid format"},
		{desc: "s length does not fit prefix width", format: "s1", data: []any{repeatDigit(300)}, wantErr: "does not fit"},
		{desc: "z rejects embedded zero byte", format: "z", data: []any{"alo\x00"}, wantErr: "contains zeros"},
		{desc: "fixed string longer than declared size", format: "c3", data: []any{"1234"}, wantErr: "longer than"},
		{desc: "X with no following option", format: "X", data: nil, wantErr: "invalid next option"},
		{desc: "X target cannot be X", format: "XXi", data: nil, wantErr: "invalid next option"},
		{desc: "X target cannot have a gap", format: "X i", data: nil, wantErr: "invalid next option"},
		{desc: "X target cannot be a string type", format: "Xci", data: nil, wantErr: "invalid next option"},
		{desc: "missing argument", format: "i", data: nil, wantErr: "expected"},
		{desc: "string where an int is expected", format: "i4", data: []any{"nope"}, wantErr: "expected number"},
		{desc: "string where a float is expected", format: "f", data: []any{"nope"}, wantErr: "expected number"},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			_, err := Pack(tc.format, tc.data...)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestUnpack(t *testing.T) {
	t.Parallel()

	t.Run("round trips every signed width with sign extension", func(t *testing.T) {
		t.Parallel()
		for size := 1; size <= 16; size++ {
			format := "i" + strconv.Itoa(size)
			packed, err := Pack(format, int64(-1))
			require.NoError(t, err)
			assert.Len(t, packed, size)

			values, pos, err := Unpack(format, packed, 0)
			require.NoError(t, err)
			assert.Equal(t, []any{int64(-1)}, values)
			assert.Equal(t, size, pos)
		}
	})

	t.Run("round trips an unsigned value through both endiannesses", func(t *testing.T) {
		t.Parallel()
		little, err := Pack("<I4", int64(0xAABBCCDD))
		require.NoError(t, err)
		values, _, err := Unpack("<I4", little, 0)
		require.NoError(t, err)
		assert.Equal(t, []any{int64(0xAABBCCDD)}, values)

		big, err := Pack(">I4", int64(0xAABBCCDD))
		require.NoError(t, err)
		assert.Equal(t, reverseString(little), big)
		values, _, err = Unpack(">I4", big, 0)
		require.NoError(t, err)
		assert.Equal(t, []any{int64(0xAABBCCDD)}, values)
	})

	t.Run("starting position is honored and returned 0-indexed past the read", func(t *testing.T) {
		t.Parallel()
		packed, err := Pack("i4i4i4i4", int64(1), int64(2), int64(3), int64(4))
		require.NoError(t, err)

		for pos := 0; pos < 16; pos += 4 {
			values, next, err := Unpack("i4", packed, pos)
			require.NoError(t, err)
			assert.Equal(t, []any{int64(pos/4 + 1)}, values)
			assert.Equal(t, pos+4, next)
		}
	})

	t.Run("alignment is computed from the absolute offset, not the read's start", func(t *testing.T) {
		t.Parallel()
		packed, err := Pack("i4i4i4i4", int64(1), int64(2), int64(3), int64(4))
		require.NoError(t, err)

		values, next, err := Unpack("!4 i4", packed, 9)
		require.NoError(t, err)
		assert.Equal(t, []any{int64(4)}, values)
		assert.Equal(t, 16, next)
	})

	t.Run("s and c and z round trip", func(t *testing.T) {
		t.Parallel()
		packed, err := Pack("s1 c3 z", "hi", "abc", "world")
		require.NoError(t, err)
		values, _, err := Unpack("s1 c3 z", packed, 0)
		require.NoError(t, err)
		assert.Equal(t, []any{"hi", "abc", "world"}, values)
	})

	t.Run("f and d round trip", func(t *testing.T) {
		t.Parallel()
		packed, err := Pack("<f <d", float64(1.5), float64(2.5))
		require.NoError(t, err)
		values, pos, err := Unpack("<f <d", packed, 0)
		require.NoError(t, err)
		assert.Equal(t, []any{1.5, 2.5}, values)
		assert.Equal(t, len(packed), pos)
	})

	t.Run("x consumes a byte and X only pads", func(t *testing.T) {
		t.Parallel()
		packed, err := Pack("b x b", int64(1), int64(2))
		require.NoError(t, err)
		values, pos, err := Unpack("b x b", packed, 0)
		require.NoError(t, err)
		assert.Equal(t, []any{int64(1), int64(2)}, values)
		assert.Equal(t, len(packed), pos)
	})
}

func TestUnpackErrors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc    string
		format  string
		data    string
		pos     int
		wantErr string
	}{
		{
			desc:    "16 byte integer overflows a Lua integer",
			format:  "i16",
			data:    "\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03",
			wantErr: "16-byte integer does not fit into Lua Integer",
		},
		{desc: "buffer shorter than the declared width", format: "i4", data: "\x01\x02", wantErr: "too short"},
		{desc: "z with no terminating zero byte", format: "z", data: "no terminator", wantErr: "unfinished string"},
		{desc: "x with no bytes left to consume", format: "x", data: "", wantErr: "too short"},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			_, _, err := Unpack(tc.format, tc.data, tc.pos)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestPacksize(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc   string
		format string
		want   int
	}{
		{desc: "short", format: "h", want: 2},
		{desc: "long", format: "l", want: 4},
		{desc: "float", format: "f", want: 4},
		{desc: "default sized int", format: "i", want: 8},
		{desc: "double", format: "d", want: 8},
		{desc: "lua_Number", format: "n", want: 8},
		{desc: "lua_Integer", format: "j", want: 8},
		{desc: "fixed string", format: "c0", want: 0},
		{desc: "padding only, alignment pulls it to the boundary", format: "!8 xXi8", want: 8},
		{desc: "alignment capped by maxalign", format: "!2 xXi8", want: 2},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got, err := Packsize(tc.format)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPacksizeErrors(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc   string
		format string
	}{
		{desc: "s is variable length", format: "s"},
		{desc: "z is variable length", format: "z"},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			_, err := Packsize(tc.format)
			require.Error(t, err)
			assert.ErrorContains(t, err, "variable-length format")
		})
	}
}

func TestPackRoundTrip(t *testing.T) {
	t.Parallel()

	out, err := Pack("<zj", "test", int64(12))
	require.NoError(t, err)
	assert.Equal(t, "test\x00\f\x00\x00\x00\x00\x00\x00\x00", out)

	format := "iii"
	data := []any{int64(12), int64(12), int64(12)}
	out, err = Pack(format, data...)
	require.NoError(t, err)
	unpackeddata, pos, err := Unpack(format, out, 0)
	require.NoError(t, err)
	assert.Equal(t, data, unpackeddata)
	assert.Equal(t, len(out), pos)
}

func repeatDigit(n int) string {
	digits := make([]byte, n)
	for i := range digits {
		digits[i] = '0'
	}
	return string(digits)
}

func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
