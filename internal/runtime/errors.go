package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tanema/luaf/internal/bytecode"
	"github.com/tanema/luaf/internal/lerrors"
	"github.com/tanema/luaf/internal/parse"
)

const (
	kindLocal   = "local"
	kindGlobal  = "global"
	kindField   = "field"
	kindUpvalue = "upvalue"
	kindMethod  = "method"
)

var nameableErrPrefixes = []string{
	"attempt to index a ",
	"attempt to call a ",
	"attempt to perform arithmetic on a ",
	"attempt to perform bitwise operation on a ",
}

func isNameableErr(msg string) bool {
	for _, prefix := range nameableErrPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}

func constStringInReg(f *frame, pc, reg int64) (string, bool) {
	for i := pc - 1; i >= 0; i-- {
		instruction := f.fn.ByteCodes[i]
		if bytecode.GetA(instruction) != reg {
			continue
		}
		if bytecode.GetOp(instruction) == bytecode.LOADK {
			if s, ok := f.fn.GetConst(bytecode.GetBx(instruction)).(string); ok {
				return s, true
			}
		}
		return "", false
	}
	return "", false
}

func keyName(f *frame, pc, keyReg int64, keyK bool) (string, bool) {
	if keyK {
		name, ok := f.fn.GetConst(keyReg).(string)
		return name, ok
	}
	return constStringInReg(f, pc, keyReg)
}

func describeRegister(f *frame, pc, reg int64) (string, string, bool) {
	if name, ok := f.fn.LocalNameAt(uint8(reg), int(pc)); ok {
		return kindLocal, name, true
	}
	for i := pc - 1; i >= 0; i-- {
		instruction := f.fn.ByteCodes[i]
		if bytecode.GetA(instruction) != reg {
			continue
		}
		switch bytecode.GetOp(instruction) {
		case bytecode.MOVE:
			return describeRegister(f, i, bytecode.GetB(instruction))
		case bytecode.GETUPVAL:
			idx := bytecode.GetB(instruction)
			if int(idx) < len(f.fn.UpIndexes) {
				return kindUpvalue, f.fn.UpIndexes[idx].Name, true
			}
			return "", "", false
		case bytecode.GETTABUP:
			idx := bytecode.GetB(instruction)
			if int(idx) < len(f.fn.UpIndexes) && f.fn.UpIndexes[idx].Name == _ENVName {
				if name, ok := keyName(f, i, bytecode.GetC(instruction), bytecode.GetK(instruction)); ok {
					return kindGlobal, name, true
				}
			}
			return "", "", false
		case bytecode.GETFIELD:
			if name, ok := keyName(f, i, bytecode.GetC(instruction), true); ok {
				if isEnvReg(f, i, bytecode.GetB(instruction)) {
					return kindGlobal, name, true
				}
				return kindField, name, true
			}
			return "", "", false
		case bytecode.GETTABLE:
			if bytecode.GetK(instruction) {
				if _, ok := f.fn.GetConst(bytecode.GetC(instruction)).(int64); ok {
					return kindField, "integer index", true
				}
			}
			if name, ok := keyName(f, i, bytecode.GetC(instruction), bytecode.GetK(instruction)); ok {
				if isEnvReg(f, i, bytecode.GetB(instruction)) {
					return kindGlobal, name, true
				}
				return kindField, name, true
			}
			return "", "", false
		case bytecode.GETI:
			return kindField, "integer index", true
		case bytecode.SELF:
			if name, ok := keyName(f, i, bytecode.GetC(instruction), true); ok {
				return kindMethod, name, true
			}
			return "", "", false
		default:
			return "", "", false
		}
	}
	return "", "", false
}

func isEnvReg(f *frame, pc, reg int64) bool {
	kind, name, ok := describeRegister(f, pc, reg)
	return ok && name == _ENVName && (kind == kindLocal || kind == kindUpvalue)
}

func (vm *VM) annotate(f *frame, reg int64, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.HasSuffix(msg, "')") {
		return err
	}
	kind, name, ok := describeRegister(f, f.pc, reg)
	if !ok {
		return err
	}
	suffix := fmt.Sprintf("%s '%s'", kind, name)
	if msg == "number has no integer representation" {
		return fmt.Errorf("number (%s) has no integer representation", suffix)
	}
	if !isNameableErr(msg) {
		return err
	}
	return fmt.Errorf("%s (%s)", msg, suffix)
}

func annotateMetamethodErr(op parse.MetaMethod, err error) error {
	if err == nil {
		return nil
	}
	var luaErr *lerrors.Error
	if !errors.As(err, &luaErr) {
		return err
	}
	msg := luaErr.Err.Error()
	if !strings.HasPrefix(msg, "attempt to call a ") || strings.HasSuffix(msg, "')") {
		return err
	}
	luaErr.Err = fmt.Errorf("%s (metamethod '%s')", msg, strings.TrimPrefix(string(op), "__"))
	return luaErr
}

func (vm *VM) annotateUpvalue(f *frame, upIdx int64, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.HasSuffix(msg, "')") || !isNameableErr(msg) || int(upIdx) >= len(f.fn.UpIndexes) {
		return err
	}
	name := f.fn.UpIndexes[upIdx].Name
	if name == _ENVName {
		return err
	}
	return fmt.Errorf("%s (upvalue '%s')", msg, name)
}

func (vm *VM) annotateArithErr(f *frame, op parse.MetaMethod, bVal any, bReg, cReg int64, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	reg := bReg
	switch {
	case strings.Contains(msg, "has no integer representation"):
		if _, ok := toIntExact(bVal); ok {
			reg = cReg
		}
	case op == parse.MetaUNM || op == parse.MetaBNot:
		reg = bReg
	default:
		if isNumber(bVal) {
			reg = cReg
		}
	}
	return vm.annotate(f, reg, err)
}

func newUserErr(vm *VM, level int, val any) error {
	var ci callInfo
	csl := int(vm.callDepth) + 1
	if csl > 0 && level > 0 && level < csl {
		ci = vm.callStack[csl-level]
	} else if csl > 0 {
		ci = vm.callStack[csl-1]
	}

	var err error
	if str, isStr := val.(string); isStr {
		if level > 0 {
			str = fmt.Sprintf("%s:%d: %s", ci.filename, ci.Line, str)
		}
		err = errors.New(str)
		val = str
	} else {
		err = fmt.Errorf("(error object is a %v value)", typeName(val))
	}

	return &lerrors.Error{
		Kind:      lerrors.UserErr,
		Filename:  ci.filename,
		Line:      ci.Line,
		Column:    ci.Column,
		Err:       err,
		Traceback: vm.formatCallstack(),
		Value:     val,
	}
}

func newRuntimeErr(vm *VM, li parse.LineInfo, err error) error {
	var luaErr *lerrors.Error
	if errors.As(err, &luaErr) {
		if luaErr.Line != 0 {
			return luaErr
		}
		luaErr.Line = li.Line
		luaErr.Column = li.Column
		if len(vm.callStack) > 0 {
			luaErr.Filename = vm.callStack[vm.callDepth].filename
		}
		return luaErr
	}
	ci := callInfo{LineInfo: li}
	if len(vm.callStack) > 0 {
		ci.filename = vm.callStack[vm.callDepth].filename
	}
	return &lerrors.Error{
		Kind:      lerrors.RuntimeErr,
		Filename:  ci.filename,
		Line:      ci.Line,
		Column:    ci.Column,
		Err:       err,
		Traceback: vm.formatCallstack(),
	}
}

func (vm *VM) formatCallstack() []string {
	parts := []string{}
	for i := range vm.callDepth {
		info := vm.callStack[i]
		if strings.HasPrefix(info.filename, "<") && strings.HasSuffix(info.filename, ">") {
			parts = append(parts, fmt.Sprintf("\t%v %v", info.filename, info.name))
		} else if strings.HasPrefix(info.name, "<") && strings.HasSuffix(info.name, ">") {
			parts = append(parts, fmt.Sprintf("\t%v %v", info.filename, info.name))
		} else {
			parts = append(parts, fmt.Sprintf("\t%v:%v: in %v", info.filename, info.Line, info.name))
		}
	}

	return parts
}
