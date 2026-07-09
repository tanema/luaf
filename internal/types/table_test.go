package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTable(t *testing.T) {
	t.Parallel()
	tbl := NewTable()
	assert.Equal(t, TblFree, tbl.Hint)
	assert.Equal(t, map[string]Definition{}, tbl.FieldDefn)
	assert.Equal(t, Any, tbl.KeyDefn)
	assert.Equal(t, Any, tbl.ValDefn)
}

func TestTableCheck(t *testing.T) {
	t.Parallel()
	cases := []struct {
		defn, other *Table
		expected    bool
	}{
		{&Table{}, &Table{}, true},
		{&Table{Hint: TblFree}, &Table{Hint: TblFree}, true},
		{&Table{Hint: TblFree}, &Table{Hint: TblStruct}, true},
		{&Table{Hint: TblStruct}, &Table{Hint: TblArray}, false},
		{&Table{Hint: TblArray, ValDefn: String}, &Table{Hint: TblArray, ValDefn: String}, true},
		{
			&Table{Hint: TblMap, KeyDefn: String, ValDefn: String},
			&Table{Hint: TblMap, KeyDefn: String, ValDefn: String},
			true,
		},
		{
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int}},
			true,
		},
		{
			&Table{Hint: TblStruct, Name: "Human", FieldDefn: map[string]Definition{"name": String, "age": Int}},
			&Table{Hint: TblStruct, Name: "Human", FieldDefn: map[string]Definition{"name": String, "age": Int}},
			true,
		},
		{
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": Int}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int}},
			false,
		},
		{
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": Int}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": String}},
			false,
		},
	}

	for i, tc := range cases {
		assert.Equal(t, tc.expected, tc.defn.Check(tc.other), "[%v] %s != %s", i, tc.defn, tc.other)
	}
}

func TestTableDiff(t *testing.T) {
	t.Parallel()
	cases := []struct {
		defn, other *Table
		expected    string
	}{
		{
			&Table{Hint: TblFree, FieldDefn: map[string]Definition{}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String}},
			"{} != {name = string}",
		}, {
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": Int}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int}},
			"{\n\t  age = int\n\t- grade = int\n\t  name = string\n}",
		}, {
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": Int}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": String}},
			"{\n\t  age = int\n\t- grade = int\n\t+ grade = string\n\t  name = string\n}",
		}, {
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int}},
			&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int, "grade": String}},
			"{\n\t  age = int\n\t  name = string\n\t+ grade = string\n}",
		},
	}

	for i, tc := range cases {
		result := tc.defn.Diff(tc.other).String()
		require.NotNil(t, result)
		assert.Equal(t, tc.expected, result, "[%v] %s does not equal %s", i, result)
	}
}

func TestTableString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		defn     *Table
		expected string
	}{
		{&Table{}, "{}"},
		{&Table{Hint: TblFree}, "{}"},
		{&Table{Hint: TblArray, ValDefn: String}, "{[string]}"},
		{&Table{Hint: TblMap, KeyDefn: String, ValDefn: String}, "{[string] = string}"},
		{&Table{Hint: TblStruct, FieldDefn: map[string]Definition{"name": String, "age": Int}}, `{age = int, name = string}`},
		{&Table{Hint: TblStruct, Name: "Human", FieldDefn: map[string]Definition{"name": String, "age": Int}}, `{Human}`},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, tc.defn.String())
	}
}
