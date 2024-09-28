package shine

type (
	exprType int
	exprDesc struct {
		kind    exprType
		a, b, c uint16
		name    string
	}
)

const (
	constExpr exprType = iota
	nilExpr
	boolExpr

	localExpr
	upvalueExpr
	indexExpr
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
