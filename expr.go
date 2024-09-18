package shine

type (
	exprType int
	exprDesc struct {
		kind  exprType
		value Value
		idx   uint16
	}
)

const (
	nilExpr exprType = iota
	booleanExpr
	integerExpr
	floatExpr
	stringExpr
	localExpr
	upvalueExpr
	indexExpr
	indexFieldExpr
	indexIntExpr
	indexUpFieldExpr
	functionExpr
	closureExpr
	callExpr
	varArgsExpr
	unaryOpExpr
	binaryOpExpr
	testExpr
	compareExpr
)
