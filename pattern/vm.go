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
		return fmt.Sprintf("SAVE %v", b.a)
	case opJmp:
		return fmt.Sprintf("JMP %v", b.a)
	case opNumber:
		return fmt.Sprintf("NUM %v", b.a)
	case opMatch:
		return fmt.Sprintf("MATCH %v", b.a)
	case opSplit:
		code = "SPLIT"
	case opBrace:
		code = "BRACE"
	}
	return fmt.Sprintf("%v %v %v", code, b.a, b.b)
}

// Simple recursive virtual machine based on the
// "Regular Expression Matching: the Virtual Machine Approach" (https://swtch.com/~rsc/regexp/regexp2.html)
func eval(src []byte, instructions []bytecode, sp int) (bool, int, []*Match, error) {
	matched, _, sp, matches, err := _eval(src, instructions, 0, sp)
	return matched, sp, matches, err
}

func _eval(src []byte, instructions []bytecode, pc, sp int) (bool, int, int, []*Match, error) {
	matches := []*Match{}
	for {
		inst := instructions[pc]
		switch inst.op {
		case opChar:
			if sp >= len(src) || !inst.class.Matches(rune(src[sp])) {
				return false, pc, sp, nil, nil
			}
			pc++
			sp++
		case opMatch:
			matched := inst.a == 0 || (inst.a > 0 && sp >= len(src))
			return matched, pc, sp, matches, nil
		case opJmp:
			pc += inst.a
		case opSplit:
			matched, npc, nsp, _, err := _eval(src, instructions, pc+inst.a, sp)
			if err != nil || matched {
				return matched, npc, nsp, nil, err
			}
			pc += inst.b
		case opSave:
			matched, npc, nsp, newMatches, err := _eval(src, instructions, pc+1, sp)
			if err != nil || !matched {
				return false, npc, nsp, nil, err
			}
			matches = append(matches, append([]*Match{{Start: sp, End: nsp, Subs: string(src[sp:nsp])}}, newMatches...)...)
			if inst.a >= 1 {
				return true, npc, nsp, matches, nil
			}
			sp = nsp
			pc = npc + 1
		case opNumber:
			// idx := inst.a * 2
			// if idx >= m.CaptureLength()-1 {
			//	return nil, fmt.Errorf("invalid capture index %v", idx)
			// }
			// capture := src[match.Start:match.End]
			// for i := 0; i < len(capture); i++ {
			//	if i+sp >= len(src) || capture[i] != src[i+sp] {
			//		return pc, sp, nil, nil
			//	}
			// }
			// pc++
			// sp += len(capture)
		default:
			return false, pc, sp, nil, fmt.Errorf("invalid operation happened while executing pattern")
		}
	}
}
