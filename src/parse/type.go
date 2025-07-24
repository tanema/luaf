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
		paramdefn []namedPairTypeDef
		retdefn   []typeDefinition
	}
	namedPairTypeDef struct {
		name string
		defn typeDefinition
	}
	mapTypeDef struct {
		keyDefn typeDefinition
		valDefn typeDefinition
	}
	arrayTypeDef struct {
		defn typeDefinition
	}
	tblTypeDef struct {
		defn map[string]typeDefinition
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
	switch tval := val.(type) {
	case *simpleType:
		return tval.name == typeNameFloat || tval.name == typeNameInt
	case *numberType:
		return true
	default:
		return false
	}
}

func (t *simpleType) Check(val any) bool {
	switch tval := val.(type) {
	case *simpleType:
		return t.name == tval.name
	default:
		return false
	}
}

func (t *simpleType) String() string { return t.name }

func (t *fnTypeDef) Check(val any) bool {
	tfn, ok := val.(*fnTypeDef)
	if !ok {
		return false
	} else if len(t.paramdefn) != len(tfn.paramdefn) || len(t.retdefn) != len(tfn.retdefn) {
		return false
	}

	for i, p := range t.paramdefn {
		if !p.defn.Check(tfn.paramdefn[i].defn) {
			return false
		}
	}

	for i, r := range t.retdefn {
		if !r.Check(tfn.retdefn[i]) {
			return false
		}
	}

	return true
}

func (t *fnTypeDef) String() string {
	params := make([]string, len(t.paramdefn))
	for i, p := range t.paramdefn {
		params[i] = fmt.Sprintf("%s:%s", p.name, p.defn.String())
	}

	var retStr string
	if len(t.retdefn) == 0 {
		retStr = "any"
	} else if len(t.retdefn) == 1 {
		retStr = t.retdefn[0].String()
	} else {
		returns := make([]string, len(t.retdefn))
		for i, r := range t.retdefn {
			returns[i] = r.String()
		}
		retStr = fmt.Sprintf("(%s)", strings.Join(returns, ", "))
	}

	return fmt.Sprintf("function(%s): %s", strings.Join(params, ", "), retStr)
}

func (t *mapTypeDef) Check(val any) bool {
	other, ok := val.(*mapTypeDef)
	if !ok {
		return false
	}
	return t.keyDefn.Check(other.keyDefn) && t.valDefn.Check(other.valDefn)
}

func (t *mapTypeDef) String() string {
	return fmt.Sprintf("{[%s]: %s}", t.keyDefn, t.valDefn)
}

func (t *arrayTypeDef) Check(val any) bool {
	other, ok := val.(*arrayTypeDef)
	if !ok {
		return false
	}
	return t.defn.Check(other.defn)
}

func (t *arrayTypeDef) String() string {
	return fmt.Sprintf("{[%s]}", t.defn)
}

func (t *tblTypeDef) Check(val any) bool {
	_, ok := val.(*exTable)
	// TODO check keys
	return ok
}

func (t *tblTypeDef) String() string { return "{TODO fields}" }
