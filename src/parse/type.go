package parse

import (
	"fmt"
	"strings"
)

type (
	typeDefinition interface {
		fmt.Stringer
		Check(val any) bool
	}
	typeUnion struct {
		defn []typeDefinition
	}
	typeIntersection struct {
		defn []typeDefinition
	}
	typeDef struct {
		nillable bool
		defn     typeDefinition
	}
	simpleType struct {
		name string
	}
	anyType    struct{}
	numberType struct{}
	fnTypeDef  struct {
		// paramdefn []typeDefinition
		// retdefn   []typeDefinition
	}
	tblTypeDef struct {
		// can be 4 types
		// - freeform natural lua data dump
		// - map key type, value type
		// - array value type
		// - struct explicit fields
		// all table types can have function definitions
	}
)

const (
	typeNameAny    = "any"
	typeNameNil    = "nil"
	typeNameString = "string"
	typeNameBool   = "bool"
	typeNameNumber = "number"
	typeNameInt    = "int"
	typeNameFloat  = "float"
	typeNameTable  = "table"
)

var (
	typeAny           = &anyType{}
	typeNil           = &simpleType{name: typeNameNil}
	typeString        = &simpleType{name: typeNameString}
	typeBool          = &simpleType{name: typeNameBool}
	typeNumber        = &numberType{}
	typeInt           = &simpleType{name: typeNameInt}
	typeFloat         = &simpleType{name: typeNameFloat}
	typeFreeformTable = &simpleType{name: typeNameTable} // table with no key restrictions

	defaultTypeDefns = map[string]typeDefinition{
		typeNameAny:    typeAny,
		typeNameNil:    typeNil,
		typeNameString: typeString,
		typeNameBool:   typeBool,
		typeNameNumber: typeNumber,
		typeNameInt:    typeInt,
		typeNameFloat:  typeFloat,
		typeNameTable:  typeFreeformTable,
	}
)

func (t *typeDef) Check(val any) bool {
	if val == nil && t.nillable {
		return true
	}
	return t.defn.Check(val)
}

func (t *typeDef) String() string {
	if t.nillable {
		return fmt.Sprintf("%s?", t.defn)
	}
	return t.defn.String()
}

func (t *typeUnion) Check(val any) bool {
	for _, defn := range t.defn {
		if defn.Check(val) {
			return true
		}
	}
	return false
}

func (t *typeUnion) String() string {
	parts := make([]string, len(t.defn))
	for i, d := range t.defn {
		parts[i] = d.String()
	}
	return strings.Join(parts, " | ")
}

func (t *typeIntersection) Check(val any) bool {
	for _, defn := range t.defn {
		if !defn.Check(val) {
			return false
		}
	}
	return true
}

func (t *typeIntersection) String() string {
	parts := make([]string, len(t.defn))
	for i, d := range t.defn {
		parts[i] = d.String()
	}
	return strings.Join(parts, " & ")
}

func (t *anyType) Check(_ any) bool  { return true }
func (t *anyType) String() string    { return "any" }
func (t *numberType) String() string { return "number" }

func (t *numberType) Check(val any) bool {
	st, isSt := val.(*simpleType)
	if !isSt {
		return false
	}
	return st.name == typeNameFloat || st.name == typeNameInt
}

func (t *simpleType) Check(val any) bool {
	st, isSt := val.(*simpleType)
	if !isSt {
		return false
	}
	return t.name == st.name
}

func (t *simpleType) String() string  { return t.name }
func (t *fnTypeDef) Check(_ any) bool { return false }
func (t *fnTypeDef) String() string   { return "function(TODO)" }

func (t *tblTypeDef) Check(val any) bool {
	_, ok := val.(*exTable)
	// TODO check keys
	return ok
}

func (t *tblTypeDef) String() string { return "{TODO fields}" }
