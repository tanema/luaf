package pattern

import (
	"errors"
	"fmt"
)

type (
	op       int
	bytecode struct {
		class class
		op    op
		a, b  int
	}
	// Match is a single substring match found in a string.
	Match struct {
		Subs  string // substring found
		Start int    // start index
		End   int    // end index
	}
)

// opChar     : If the character SP points at is not c, stop this thread: it failed.
//              Otherwise, advance SP to the next character and advance PC to the
//              next instruction.
// opMatch    : Stop this thread: it found a match.
// opJmp      : Jump to (set the PC to point at) the instruction at x.
// opSplit    : Split execution. Create a new thread with SP copied from the current
//              thread. One thread continues with PC x. The other continues with PC
//              y. (Like a simultaneous jump to both locations.)
// opSave     : Similar to split but saves the current string pointer in the ith
//              slot in the saved pointer array for the current thread
// opNumber   : match against capture group %n
// opFrontier : zero width assertion; matches when SP-1 is not in the class but
//              SP is (out of bounds counts as a NUL byte, as in Lua)

const (
	opChar op = iota
	opMatch
	opJmp
	opSplit
	opSave
	opNumber
	opFrontier
)

func (b *bytecode) String() string {
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
		return fmt.Sprintf("SPLIT %v %v", b.a, b.b)
	case opFrontier:
		return fmt.Sprintf("FRONTIER %s", b.class)
	}
	return "UNKNOWN"
}

// Simple recursive virtual machine based on the
// "Regular Expression Matching: the Virtual Machine Approach" (https://swtch.com/~rsc/regexp/regexp2.html)
//
// src is matched as raw bytes rather than runes. Lua strings (and the patterns
// matched against them) are byte sequences, not necessarily valid UTF-8, so the
// indices in Match must line up with Go's native (byte based) string indexing.
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
			matched, npc, nsp, subMatches, err := _eval(src, instructions, pc+inst.a, sp)
			if err != nil || matched {
				return matched, npc, nsp, subMatches, err
			}
			pc += inst.b
		case opSave:
			matched, npc, nsp, newMatches, err := _eval(src, instructions, pc+1, sp)
			if err != nil || !matched {
				return false, npc, nsp, nil, err
			}
			if sp != nsp {
				matches = append(matches, &Match{Start: sp, End: nsp, Subs: string(src[sp:nsp])})
			}
			matches = append(matches, newMatches...)
			if inst.a >= 1 { // Root save group
				return true, npc, nsp, matches, nil
			}
			sp = nsp
			pc = npc + 1
		case opNumber:
			idx := inst.a * 2
			if idx >= len(matches) {
				return false, pc, sp, nil, fmt.Errorf("invalid capture index %v", idx)
			}
			capture := matches[idx].Subs
			for i := 0; i < len(capture); i++ {
				if i+sp >= len(src) || capture[i] != src[i+sp] {
					return false, pc, sp, nil, nil
				}
			}
			pc++
			sp += len(capture)
		case opFrontier:
			var prev, cur rune
			if sp > 0 {
				prev = rune(src[sp-1])
			}
			if sp < len(src) {
				cur = rune(src[sp])
			}
			if inst.class.Matches(prev) || !inst.class.Matches(cur) {
				return false, pc, sp, nil, nil
			}
			pc++
		default:
			return false, pc, sp, nil, errors.New("invalid operation happened while executing pattern")
		}
	}
}
