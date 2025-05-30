// Package types contains definitions for types for a future type system.
package types

// Type aliases
// optional   => union(val | nil)
// lua tables => union(hashtable & array).
type (
	// Any could be anything.
	Any struct{}
	// String matches any string.
	String struct{}
	// Bool matches any boolean value.
	Bool struct{}
	// Nil matches a nil value.
	Nil struct{}
	// Integer matches integer numbers.
	Integer struct{}
	// Float matches float numbers.
	Float struct{}
	// Number matches any number, Integer or Float.
	Number struct{}
	// Function matches a function with a FnSignature.
	Function struct{ FnSignature }
	// HashTable is a table with only key values.
	HashTable struct{ Def map[string]any }
	// Array is a table with only number indexes.
	Array struct{ Def any }
	// Union is a type union between two or more types.
	Union struct{ Types []any }
	// Intersection is a type intersection between two or more types.
	Intersection struct{ Types []any }
	// Literal will match a literal value, so if the type can be "CONNECTED" | "OFF"
	// it match those strings exactly no any string.
	Literal[T any] struct{ Val T }
	// FnSignature captures a call signature for a function.
	FnSignature struct {
		Params map[string]any
		Return any
	}
)
