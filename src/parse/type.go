package parse

type (
	TypeDefinition interface {
		Check(any) bool
	}
	TypeUnion struct {
		defn []TypeDefinition
	}
	TypeIntersection struct {
		defn []TypeDefinition
	}
	TypeDef struct {
		optional bool
		defn     TypeDefinition
	}
	SimpleType struct {
		fn func(any) bool
	}
)

var (
	typeAny    = &SimpleType{fn: func(any) bool { return true }}
	typeNil    = &SimpleType{fn: func(val any) bool { return val == nil }}
	typeString = &SimpleType{fn: isAString}
	typeBool   = &SimpleType{fn: isABool}
	typeNumber = &SimpleType{fn: isANumber}
	typeInt    = &SimpleType{fn: isAnInt}
	typeFloat  = &SimpleType{fn: isAFloat}
)

func (t *TypeDef) Check(val any) bool {
	if val == nil && t.optional {
		return true
	}
	return t.defn.Check(val)
}

func (t *TypeUnion) Check(val any) bool {
	for _, defn := range t.defn {
		if defn.Check(val) {
			return true
		}
	}
	return false
}

func (t *TypeIntersection) Check(val any) bool {
	for _, defn := range t.defn {
		if !defn.Check(val) {
			return false
		}
	}
	return true
}

func (t *SimpleType) Check(val any) bool {
	return t.fn(val)
}

func isANumber(val any) bool {
	switch val.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func isAnInt(val any) bool {
	_, ok := val.(int64)
	return ok
}

func isAFloat(val any) bool {
	_, ok := val.(float64)
	return ok
}

func isAString(val any) bool {
	_, ok := val.(string)
	return ok
}

func isABool(val any) bool {
	_, ok := val.(bool)
	return ok
}
