package types

import (
	"fmt"
	"slices"
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
	// TableDiff captures the difference between two table definitons to help identify
	// ways to fix issues.
	TableDiff struct {
		A, B           *Table
		ExtraFields    []string
		MissingFields  []string
		ConflictFields map[string][2]Definition
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
func (t *Table) Check(val Definition) bool { return Equal(t, val) }

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

// Diff will generate a TableDiff of two table type definitions to allow easy fixing.
func (t *Table) Diff(other *Table) *TableDiff {
	// hint catches the fact that free tables match structs
	if Equal(t, other) && t.Hint == other.Hint {
		return nil
	}

	diff := &TableDiff{A: t, B: other, ConflictFields: map[string][2]Definition{}}
	if t.Hint != other.Hint || t.Hint == TblArray || t.Hint == TblMap {
		return diff
	}

	for key, valDefn := range t.FieldDefn {
		otherValDef, hasKey := other.FieldDefn[key]
		if !hasKey {
			diff.MissingFields = append(diff.MissingFields, key)
		} else if !valDefn.Check(otherValDef) {
			diff.ConflictFields[key] = [2]Definition{valDefn, otherValDef}
		}
	}
	for key := range other.FieldDefn {
		_, hasKey := t.FieldDefn[key]
		if !hasKey {
			diff.ExtraFields = append(diff.ExtraFields, key)
		}
	}
	return diff
}

func (diff *TableDiff) String() string {
	if diff.A.Hint != diff.B.Hint || diff.A.Hint == TblArray || diff.A.Hint == TblMap {
		return fmt.Sprintf("%s != %s", diff.A, diff.B)
	}
	keys := []string{}
	for key := range diff.A.FieldDefn {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	parts := []string{}
	for _, key := range keys {
		defn := diff.A.FieldDefn[key]
		if slices.Contains(diff.MissingFields, key) {
			parts = append(parts, fmt.Sprintf("\t- %s = %s", key, defn))
		} else if conflict, hasConflict := diff.ConflictFields[key]; hasConflict {
			parts = append(parts, fmt.Sprintf("\t- %s = %s", key, conflict[0]))
			parts = append(parts, fmt.Sprintf("\t+ %s = %s", key, conflict[1]))
		} else {
			parts = append(parts, fmt.Sprintf("\t  %s = %s", key, defn))
		}
	}
	for _, key := range diff.ExtraFields {
		parts = append(parts, fmt.Sprintf("\t+ %s = %s", key, diff.B.FieldDefn[key]))
	}

	return fmt.Sprintf("{\n%s\n}", strings.Join(parts, "\n"))
}

func (t *Table) String() string {
	switch t.Hint {
	case TblArray:
		return fmt.Sprintf("{[%s]}", t.ValDefn.String())
	case TblMap:
		return fmt.Sprintf("{[%s] = %s}", t.KeyDefn.String(), t.ValDefn.String())
	case TblStruct:
		if t.Name != "" {
			return fmt.Sprintf("{%s}", t.Name)
		}
		keys := []string{}
		for key := range t.FieldDefn {
			keys = append(keys, key)
		}
		slices.Sort(keys)

		parts := []string{}
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s = %s", key, t.FieldDefn[key].String()))
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
	case TblFree:
		fallthrough
	default:
		return "{}"
	}
}
