package types

import (
	"fmt"
	"strings"
)

type (
	// Definition is a general interface for all type definitions.
	Definition interface {
		fmt.Stringer
		Check(val Definition) bool
	}
	// Union describes a type that can match multiple types.
	Union struct{ Defn []Definition }
	// Intersection describes a type that combines multiple types.
	Intersection struct{ Defn []Definition }
	// Simple describes a builtin type handled by the parser.
	Simple  struct{ Name string }
	anyType struct{}
	// Function describes a function type with specified params and returns.
	Function struct {
		Params []NamedPair
		Return []Definition
	}
	// NamedPair captures a single param for a function.
	NamedPair struct {
		Name string
		Defn Definition
	}
)

const (
	// NameAny is a label for the any type.
	NameAny = "any"
	// NameNil is a label for the nil type.
	NameNil = "nil"
	// NameString is a label for the string type.
	NameString = "string"
	// NameBool is a label for the bool type.
	NameBool = "bool"
	// NameNumber is a label for the number type.
	NameNumber = "number"
	// NameInt is a label for the int type.
	NameInt = "int"
	// NameFloat is a label for the float type.
	NameFloat = "float"
	// NameTable is a label for the table type.
	NameTable = "table"
)

var (
	// Any is a type any to match against.
	Any = &anyType{}
	// Number is a type number to match against.
	Number = &Union{Defn: []Definition{Int, Float}}
	// Nil is a type nil to match against.
	Nil = &Simple{Name: NameNil}
	// String is a type string to match against.
	String = &Simple{Name: NameString}
	// Bool is a type bool to match against.
	Bool = &Simple{Name: NameBool}
	// Int is a type int to match against.
	Int = &Simple{Name: NameInt}
	// Float is a type float to match against.
	Float = &Simple{Name: NameFloat}
	// AnyTable is a default type that is the most flexible table type.
	AnyTable = &Table{Hint: TblFree}
	// DefaultDefns is a collection of types that exist by default.
	DefaultDefns = map[string]Definition{
		NameAny:    Any,
		NameNumber: Number,
		NameNil:    Nil,
		NameString: String,
		NameBool:   Bool,
		NameInt:    Int,
		NameFloat:  Float,
		NameTable:  AnyTable,
	}
)

// Check will check if this type definition matches another.
func (t *Union) Check(val Definition) bool { return Equal(t, val) }
func (t *Union) String() string            { return fmt.Sprintf("{%s}", fmtDefns(t.Defn, " | ")) }

// Check will check if this type definition matches another.
func (t *Intersection) Check(val Definition) bool { return Equal(t, val) }
func (t *Intersection) String() string            { return fmt.Sprintf("{%s}", fmtDefns(t.Defn, " & ")) }

// Check will check if this type definition matches another.
func (t *anyType) Check(_ Definition) bool { return true }
func (t *anyType) String() string          { return "any" }

// Check will check if this type definition matches another.
func (t *Simple) Check(val Definition) bool { return Equal(t, val) }
func (t *Simple) String() string            { return t.Name }

// Check will check if this type definition matches another.
func (t *Function) Check(val Definition) bool { return Equal(t, val) }

func (t *Function) String() string {
	params := make([]Definition, len(t.Params))
	for i, p := range t.Params {
		params[i] = &p
	}

	var retStr string
	if len(t.Return) == 0 {
		retStr = "any"
	} else if len(t.Return) == 1 {
		retStr = t.Return[0].String()
	} else {
		retStr = fmt.Sprintf("(%s)", fmtDefns(t.Return, ", "))
	}

	return fmt.Sprintf("function(%s): %s", fmtDefns(params, ", "), retStr)
}

// Check will check if this type definition matches another.
func (t *NamedPair) Check(val Definition) bool { return t.Defn.Check(val) }
func (t *NamedPair) String() string            { return fmt.Sprintf("%s: %s", t.Name, t.Defn.String()) }

// Equal compares two types and will return if they are equal or rather if B matches the
// definition laid out by a.
func Equal(a, b Definition) bool {
	if a == b {
		return true
	}

	switch ta := a.(type) {
	case *anyType:
		return true
	case *Union:
		for _, defn := range ta.Defn {
			if Equal(defn, b) {
				return true
			}
		}
		return false
	case *Intersection:
		for _, defn := range ta.Defn {
			if !Equal(defn, b) {
				return false
			}
		}
		return true
	case *Simple:
		other, isSimple := b.(*Simple)
		return isSimple && ta.Name == other.Name
	case *Function:
		other, isFn := b.(*Function)
		if !isFn || len(ta.Params) != len(other.Params) || len(ta.Return) != len(other.Return) {
			return false
		}
		for i, p := range ta.Params {
			if p.Defn != other.Params[i].Defn {
				return false
			}
		}
		for i, r := range ta.Return {
			if r != other.Return[i] {
				return false
			}
		}
		return true
	case *Table:
		other, isTbl := b.(*Table)
		if !isTbl {
			return false
		}
		switch ta.Hint {
		case TblMap:
			return other.Hint == TblMap &&
				ta.KeyDefn.Check(other.KeyDefn) &&
				ta.ValDefn.Check(other.ValDefn) &&
				ta.checkFields(other)
		case TblArray:
			return other.Hint == TblArray &&
				ta.ValDefn.Check(other.ValDefn) &&
				ta.checkFields(other)
		case TblStruct:
			if other.Hint == TblArray {
				return false
			}
			return ta.checkFields(other)
		}
		return true
	default:
		return false
	}
}

// Reduce will boil down a bunch of definitions to a single definition. If there
// is nothing that unifies the kinds, then Any will be returned.
func Reduce(defns []Definition) Definition {
	if defns = Unique(defns); len(defns) == 1 {
		return defns[0]
	} else if len(defns) == 2 && contains(defns, []Definition{Float, Int}) {
		return Number
	}
	return Any
}

func contains(fullSet, has []Definition) bool {
	for _, defn := range fullSet {
		for _, h := range has {
			if defn == h {
				return true
			}
		}
	}
	return false
}

// Unique will remove any duplicate definitions in the array.
func Unique(defns []Definition) []Definition {
	seen := map[Definition]int{}
	result := []Definition{}
	for _, defn := range defns {
		if _, ok := seen[defn]; !ok {
			seen[defn] = 1
			result = append(result, defn)
		}
	}
	return result
}

func fmtDefns(defn []Definition, sep string) string {
	parts := make([]string, len(defn))
	for i, d := range defn {
		parts[i] = d.String()
	}
	return strings.Join(parts, sep)
}
