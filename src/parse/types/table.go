package types

import (
	"fmt"
	"strings"
)

type (
	// TableHint is a hint added to the table struct to define how the type is handled
	// in iterators and type checking.
	TableHint int
	// Table is the lua table type description which can act like a struct, array
	// map, and class.
	Table struct {
		// name can really only be used on structs because if the table is free, array
		// or map, then they are general unnamed types that are still equal if the
		// inner types match. However if they are these simple types with methods
		// defined on them, then more accurate type checking has to happen because
		// they become more refined types.
		Name string
		// hint gives an idea of how the table is used for accurate and faster type checking.
		Hint    TableHint
		KeyDefn Definition
		ValDefn Definition
		// fieldDefn is for struct definitions as well as methods defined on the
		// other types. When the type is an array, it can still have methods defined
		// on it.
		FieldDefn map[string]Definition
	}
)

const (
	// TblFree is the hint to describe that the table has no defined fields or use.
	TblFree TableHint = iota
	// TblMap is the hint the describe a table with key/value stored in them.
	TblMap
	// TblArray is the hint for tables that are index by integers.
	TblArray
	// TblStruct is the hint for table with strict attribute definition.
	TblStruct
)

// NewTable creates a new table type definition starting off as a freeform table.
func NewTable() *Table {
	return &Table{
		Hint:      TblFree,
		KeyDefn:   Any,
		ValDefn:   Any,
		FieldDefn: map[string]Definition{},
	}
}

// Check will check if this type definition matches another.
func (t *Table) Check(val any) bool {
	other, ok := val.(*Table)
	if !ok {
		return false
	}

	switch t.Hint {
	case TblArray, TblMap:
		if t.Hint != other.Hint {
			return false
		} else if len(t.FieldDefn) == 0 && len(other.FieldDefn) == 0 {
			return t.checkKV(other) // not a refined map or array so it can match any map or array with same types
		}
		// map or array with methods on it to extend functionality so we need to
		// do a check with higher resolution
		return t.checkKV(other) && t.checkFields(other)
	case TblStruct:
		if other.Hint != TblStruct && other.Hint != TblFree {
			return false
		}
		return t.checkFields(other)
	}
	return true
}

func (t *Table) checkKV(other *Table) bool {
	return t.KeyDefn.Check(other.KeyDefn) && t.ValDefn.Check(other.ValDefn)
}

func (t *Table) checkFields(other *Table) bool {
	if t.Name != other.Name || len(t.FieldDefn) != len(other.FieldDefn) {
		return false
	}
	for key, valDefn := range t.FieldDefn {
		other, hasKey := other.FieldDefn[key]
		if !hasKey || !valDefn.Check(other) {
			return false
		}
	}
	return true
}

// generate key diff between two table descriptions
// missing: the keys that are not in the original table but not in the other
// extra: the keys in the other table that are not in the original
// conflicts: the values that are different types under the same key
func (t *Table) diff(other *Table) ([]string, []string, map[string][2]Definition) { //nolint:unused
	extra := []string{}
	missing := []string{}
	conflicts := map[string][2]Definition{}
	for key, valDefn := range t.FieldDefn {
		otherValDef, hasKey := other.FieldDefn[key]
		if !hasKey {
			missing = append(missing, key)
		} else if !valDefn.Check(otherValDef) {
			conflicts[key] = [2]Definition{valDefn, otherValDef}
		}
	}
	for key := range other.FieldDefn {
		_, hasKey := t.FieldDefn[key]
		if !hasKey {
			extra = append(extra, key)
		}
	}
	return missing, extra, conflicts
}

func (t *Table) String() string {
	switch t.Hint {
	case TblArray:
		return fmt.Sprintf("{[%s]}", t.ValDefn.String())
	case TblMap:
		return fmt.Sprintf("{[%s] = %s}", t.KeyDefn.String(), t.ValDefn.String())
	case TblStruct:
		parts := []string{}
		for key, valDefn := range t.FieldDefn {
			parts = append(parts, fmt.Sprintf("%s: %s", key, valDefn.String()))
		}
		return fmt.Sprintf("{\n%s\n}", strings.Join(parts, "\n"))
	case TblFree:
		fallthrough
	default:
		return "{}"
	}
}
