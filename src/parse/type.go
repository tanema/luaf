package parse

type (
	typeDefinition interface {
		Check(val any) bool
	}
	typeUnion struct {
		defn []typeDefinition
	}
	typeIntersection struct {
		defn []typeDefinition
	}
	typeDef struct {
		optional bool
		defn     typeDefinition
	}
	simpleType struct {
		fn func(any) bool
	}
	fnTypeDef struct {
		fn *FnProto
	}
)

var (
	typeAny    = &simpleType{fn: func(any) bool { return true }}
	typeNil    = &simpleType{fn: func(val any) bool { return val == nil }}
	typeString = &simpleType{fn: isAString}
	typeBool   = &simpleType{fn: isABool}
	typeNumber = &simpleType{fn: isANumber}
	typeInt    = &simpleType{fn: isAnInt}
	typeFloat  = &simpleType{fn: isAFloat}
)

func (t *typeDef) Check(val any) bool {
	if val == nil && t.optional {
		return true
	}
	return t.defn.Check(val)
}

func (t *typeUnion) Check(val any) bool {
	for _, defn := range t.defn {
		if defn.Check(val) {
			return true
		}
	}
	return false
}

func (t *typeIntersection) Check(val any) bool {
	for _, defn := range t.defn {
		if !defn.Check(val) {
			return false
		}
	}
	return true
}

func (t *simpleType) Check(val any) bool {
	return t.fn(val)
}

func (t *fnTypeDef) Check(_ any) bool {
	return false
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
