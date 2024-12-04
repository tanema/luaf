package pattern

import "fmt"

type (
	Op       int
	bytecode struct {
		op    Op
		class class
		a, b  int
	}
	Match struct {
		Subs  string
		Start int
		End   int
	}
)

const (
	opChar   Op = iota // If the character SP points at is not c, stop this thread: it failed. Otherwise, advance SP to the next character and advance PC to the next instruction.
	opMatch            // Stop this thread: it found a match.
	opJmp              // Jump to (set the PC to point at) the instruction at x.
	opSplit            // Split execution. Create a new thread with SP copied from the current thread. One thread continues with PC x. The other continues with PC y. (Like a simultaneous jump to both locations.)
	opSave             // Similar to split but saves the current string pointer in the ith slot in the saved pointer array for the current thread
	opBrace            // match against %b brace op
	opNumber           // match against capture group %n
)

func (b *bytecode) String() string {
	var code string
	switch b.op {
	case opChar:
		return fmt.Sprintf("CHAR %s", b.class)
	case opSave:
		return "SAVE"
	case opJmp:
		return fmt.Sprintf("JMP %v", b.a)
	case opNumber:
		return fmt.Sprintf("NUM %v", b.a)
	case opMatch:
		return "MATCH"
	case opSplit:
		code = "SPLIT"
	case opBrace:
		code = "BRACE"
	}
	return fmt.Sprintf("%v %v %v", code, b.a, b.b)
}

// Simple recursive virtual machine based on the
// "Regular Expression Matching: the Virtual Machine Approach" (https://swtch.com/~rsc/regexp/regexp2.html)
func eval(src []byte, instructions []bytecode, pc, sp int) (*Match, error) {
	match := &Match{Start: sp}
	for {
		inst := instructions[pc]
		switch inst.op {
		case opMatch:
			match.End = sp
			if inst.a > 0 && sp >= len(src) {
				if sp >= len(src) {
					return match, nil
				}
				return nil, nil
			}
			return match, nil
		case opJmp:
			pc += inst.a
		case opChar:
			if sp >= len(src) || !inst.class.Matches(rune(src[sp])) {
				return nil, nil
			}
			pc++
			sp++
		case opSplit:
			if ms, err := eval(src, instructions, pc+inst.a, sp); err != nil || ms != nil {
				return ms, err
			}
			pc += inst.b
		case opSave:
			return eval(src, instructions, pc+1, sp)
		case opBrace:
			if sp >= len(src) || int(src[sp]) != inst.a {
				return nil, nil
			}
			count := 1
			for sp = sp + 1; sp < len(src); sp++ {
				if int(src[sp]) == inst.b {
					count--
				}
				if count == 0 {
					pc++
					sp++
					continue
				}
				if int(src[sp]) == inst.b {
					count++
				}
			}
			return nil, nil
		case opNumber:
			// idx := inst.a * 2
			// if idx >= m.CaptureLength()-1 {
			//	return nil, fmt.Errorf("invalid capture index %v", idx)
			// }
			capture := src[match.Start:match.End]
			for i := 0; i < len(capture); i++ {
				if i+sp >= len(src) || capture[i] != src[i+sp] {
					return nil, nil
				}
			}
			pc++
			sp += len(capture)
		default:
			return nil, fmt.Errorf("invalid operation happened while executing pattern")
		}
	}
}
