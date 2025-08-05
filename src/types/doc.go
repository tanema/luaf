// Package types contains all the structures used to define types, and check them
// against each other. All Check() methods should expect simple types rather than
// unions or intersections. This is because checking is done at the time of assignment
// or on expressions where the types are inferred from the data. This inferral
// will not infer a general type but a concrete type instead. So all checking
// to simplify the code should only be done With General type checking on a
// concrete type.
// One sidenote is that the parser, if the type is not specified during assignment
// it will generalize an int|float to number but this is the only generalization
// that happens and check is not called in the case where no type was declared
// anyhow.
package types //nolint:revive
