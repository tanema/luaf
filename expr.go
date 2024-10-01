package shine

type (
	exprType int
	exprDesc struct {
		kind   exprType
		name   string
		a      uint16
		b      uint16
		bConst bool
		c      uint16
		cConst bool
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
