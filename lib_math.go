package luaf

import (
	"math"
	"math/rand"
	"time"
)

var randSource = rand.New(rand.NewSource(time.Now().Unix()))

func createMathLib() *Table {
	return &Table{
		hashtable: map[any]Value{
			"huge":       &Float{val: math.MaxFloat64},
			"maxinteger": &Integer{val: math.MaxInt64},
			"mininteger": &Integer{val: math.MinInt64},
			"pi":         &Float{val: math.Pi},
			"abs":        stdMathFn("abs", false, math.Abs),
			"acos":       stdMathFn("acos", true, math.Acos),
			"asin":       stdMathFn("asin", true, math.Asin),
			"atan":       stdMathFn("atan", true, math.Atan),
			"cos":        stdMathFn("cos", true, math.Cos),
			"exp":        stdMathFn("exp", true, math.Exp),
			"sin":        stdMathFn("sin", true, math.Sin),
			"tan":        stdMathFn("tan", true, math.Tan),
			"log":        stdMathFn("log", true, math.Log),
			"sqrt":       stdMathFn("sqrt", true, math.Sqrt),
			"ceil":       stdMathFn("ceil", false, math.Ceil),
			"floor":      stdMathFn("floor", false, math.Floor),
			"deg":        stdMathFn("deg", true, mathDeg),
			"rad":        stdMathFn("rad", true, mathRad),
			"fmod":       Fn("math.fmod", stdMathFmod),
			"modf":       Fn("math.modf", stdMathModf),
			"max":        Fn("math.max", stdMathMax),
			"min":        Fn("math.min", stdMathMin),
			"random":     Fn("math.random", stdMathRandom),
			"randomseed": Fn("math.randomseed", stdMathRandomSeed),
			"tointeger":  Fn("math.tointeger", stdMathToInteger),
			"type":       Fn("math.type", stdMathType),
			"ult":        Fn("math.ult", stdMathUlt),
		},
	}
}

func stdMathFn(name string, mustFloat bool, fn func(float64) float64) *GoFunc {
	return &GoFunc{
		name: "math." + name,
		val: func(_ *VM, args []Value) ([]Value, error) {
			if err := assertArguments(args, "math."+name, "number"); err != nil {
				return nil, err
			}
			num := toFloat(args[0])
			var res Value = &Float{val: fn(num)}
			if _, isInt := args[0].(*Integer); isInt && !mustFloat {
				res = &Integer{val: toInt(res)}
			}
			return []Value{res}, nil
		},
	}
}

func stdMathFmod(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.fmod", "number"); err != nil {
		return nil, err
	}
	n, frac := math.Modf(toFloat(args[0]))
	var res Value = &Float{val: n}
	if _, isInt := args[0].(*Integer); isInt {
		res = &Integer{val: toInt(res)}
	}
	return []Value{res, &Float{val: frac}}, nil
}

func stdMathModf(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.modf", "number", "number"); err != nil {
		return nil, err
	}
	n := math.Mod(toFloat(args[0]), toFloat(args[1]))
	var res Value = &Float{val: n}
	if _, isInt := args[0].(*Integer); isInt {
		res = &Integer{val: toInt(res)}
	}
	return []Value{res}, nil
}

func stdMathMax(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.max", "number", "number"); err != nil {
		return nil, err
	}
	n := math.Max(toFloat(args[0]), toFloat(args[1]))
	var res Value = &Float{val: n}
	if _, isInt := args[0].(*Integer); isInt {
		res = &Integer{val: toInt(res)}
	}
	return []Value{res}, nil
}

func stdMathMin(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.min", "number", "number"); err != nil {
		return nil, err
	}
	n := math.Min(toFloat(args[0]), toFloat(args[1]))
	var res Value = &Float{val: n}
	if _, isInt := args[0].(*Integer); isInt {
		res = &Integer{val: toInt(res)}
	}
	return []Value{res}, nil
}

func stdMathRandomSeed(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.randomseed", "~number"); err != nil {
		return nil, err
	}
	var x int64

	if len(args) == 0 {
		x = time.Now().Unix()
	} else {
		x = toInt(args[0])
	}

	randSource.Seed(x)
	return []Value{&Integer{val: x}, &Integer{}}, nil
}

func stdMathRandom(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.random", "~number", "~number"); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return []Value{&Float{val: randSource.Float64()}}, nil
	}
	start := int64(1)
	end := toInt(args[0])
	if len(args) > 1 {
		start = end
		end = toInt(args[1])
	}
	return []Value{&Integer{val: start + randSource.Int63n(end-start)}}, nil
}

func stdMathToInteger(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.tointeger", "value"); err != nil {
		return nil, err
	}
	return []Value{&Integer{val: toInt(args[0])}}, nil
}

func stdMathType(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.type", "value"); err != nil {
		return nil, err
	}
	switch args[0].(type) {
	case *Integer:
		return []Value{&String{val: "integer"}}, nil
	case *Float:
		return []Value{&String{val: "float"}}, nil
	default:
		return []Value{&Nil{}}, nil
	}
}

func stdMathUlt(_ *VM, args []Value) ([]Value, error) {
	if err := assertArguments(args, "math.ult", "number", "number"); err != nil {
		return nil, err
	}
	a, b := toInt(args[0]), toInt(args[1])
	return []Value{&Boolean{val: a < b}}, nil
}

func mathDeg(x float64) float64 {
	return x * 180 / math.Pi
}

func mathRad(x float64) float64 {
	return x * math.Pi / 180
}
