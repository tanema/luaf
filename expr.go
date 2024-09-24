package shine

type (
	exprType int
	exprDesc struct {
		kind    exprType
		a, b, c uint16
	}
)

const (
	constExpr exprType = iota
	nilExpr
	boolExpr
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
