package shine

import "fmt"

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

func (expr *exprDesc) load(name string, res *ParseResult) {
	scope := res.Blocks[len(res.Blocks)-1]
	var inst Bytecode
	switch expr.kind {
	case stringExpr, floatExpr, integerExpr:
		ksrc := scope.addConst(expr.value)
		inst = AsBytecode(LOADK, res.sp, ksrc)
	case booleanExpr:
		asBool := expr.value.(*Boolean)
		if asBool.Val() == true {
			inst = AsBytecode(LOADBOOL, res.sp, 1)
		} else {
			inst = AsBytecode(LOADBOOL, res.sp, 0)
		}
	case nilExpr:
		inst = AsBytecode(LOADNIL, res.sp)
	default:
		panic(fmt.Sprintf("%v is not yet supported", expr.kind))
	}
	scope.ByteCodes = append(scope.ByteCodes, inst)
	scope.Locals[name] = res.sp
	res.sp++
}
