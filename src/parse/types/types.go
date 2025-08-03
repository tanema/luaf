package types

import (
	"fmt"
	"strings"
)

type (
	// Definition is a general interface for all type definitions.
	Definition interface {
		fmt.Stringer
		Check(val any) bool
	}
	// Union describes a type that can match multiple types.
	Union struct{ Defn []Definition }
	// Intersection describes a type that combines multiple types.
	Intersection struct{ Defn []Definition }
	// Optional describes a nullable type.
	Optional struct{ Defn Definition }
	// Simple describes a builtin type handled by the parser.
	Simple  struct{ Name string }
	anyType struct{}
	num     struct{}
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
	Number = &num{}
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
	// DefaultDefns is a collection of types that exist by default.
	DefaultDefns = map[string]Definition{
		NameAny:    Any,
		NameNumber: Number,
		NameNil:    Nil,
		NameString: String,
		NameBool:   Bool,
		NameInt:    Int,
		NameFloat:  Float,
	}
)

// Check will check if this type definition matches another.
func (t *Optional) Check(val any) bool {
	if val == nil {
		return true
	}
	return t.Defn.Check(val)
}

func (t *Optional) String() string {
	return fmt.Sprintf("%s?", t.Defn)
}

// Check will check if this type definition matches another.
func (t *Union) Check(val any) bool {
	for _, defn := range t.Defn {
		if defn.Check(val) {
			return true
		}
	}
	return false
}

func (t *Union) String() string {
	parts := make([]string, len(t.Defn))
	for i, d := range t.Defn {
		parts[i] = d.String()
	}
	return strings.Join(parts, " | ")
}

// Check will check if this type definition matches another.
func (t *Intersection) Check(val any) bool {
	for _, defn := range t.Defn {
		if !defn.Check(val) {
			return false
		}
	}
	return true
}

func (t *Intersection) String() string {
	parts := make([]string, len(t.Defn))
	for i, d := range t.Defn {
		parts[i] = d.String()
	}
	return strings.Join(parts, " & ")
}

// Check will check if this type definition matches another.
func (t *anyType) Check(_ any) bool { return true }
func (t *anyType) String() string   { return "any" }
func (t *num) String() string       { return "number" }

// Check will check if this type definition matches another.
func (t *num) Check(val any) bool {
	switch tval := val.(type) {
	case *Simple:
		return tval.Name == NameFloat || tval.Name == NameInt
	case *num:
		return true
	default:
		return false
	}
}

// Check will check if this type definition matches another.
func (t *Simple) Check(val any) bool {
	switch tval := val.(type) {
	case *Simple:
		return t.Name == tval.Name
	default:
		return false
	}
}

func (t *Simple) String() string { return t.Name }

// Check will check if this type definition matches another.
func (t *Function) Check(val any) bool {
	tfn, ok := val.(*Function)
	if !ok {
		return false
	} else if len(t.Params) != len(tfn.Params) || len(t.Return) != len(tfn.Return) {
		return false
	}

	for i, p := range t.Params {
		if !p.Defn.Check(tfn.Params[i].Defn) {
			return false
		}
	}

	for i, r := range t.Return {
		if !r.Check(tfn.Return[i]) {
			return false
		}
	}

	return true
}

func (t *Function) String() string {
	params := make([]string, len(t.Params))
	for i, p := range t.Params {
		params[i] = fmt.Sprintf("%s:%s", p.Name, p.Defn.String())
	}

	var retStr string
	if len(t.Return) == 0 {
		retStr = "any"
	} else if len(t.Return) == 1 {
		retStr = t.Return[0].String()
	} else {
		returns := make([]string, len(t.Return))
		for i, r := range t.Return {
			returns[i] = r.String()
		}
		retStr = fmt.Sprintf("(%s)", strings.Join(returns, ", "))
	}

	return fmt.Sprintf("function(%s): %s", strings.Join(params, ", "), retStr)
}
