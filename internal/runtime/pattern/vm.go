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
	// Match is a single capture (or the whole match, at index 0) found in a string.
	// A position capture ("()") has no text: IsPosition is set and Start (== End)
	// holds the 0-based byte offset the caller should report as a 1-based position.
	Match struct {
		Subs       string // substring found
		Start      int    // start index
		End        int    // end index
		IsPosition bool
	}
)

// opChar      : If the character SP points at is not c, stop this thread: it failed.
//               Otherwise, advance SP to the next character and advance PC to the
//               next instruction.
// opMatch     : Stop this thread: it found a match.
// opJmp       : Jump to (set the PC to point at) the instruction at x.
// opSplit     : Split execution. Create a new thread with SP copied from the current
//               thread. One thread continues with PC x. The other continues with PC
//               y. (Like a simultaneous jump to both locations.)
// opSaveStart : Record SP as the start of capture slot a. Falls through to the next
//               instruction in the same program: unlike a capture group being its
//               own nested sub-match, this keeps captures fully transparent to
//               backtracking, so a quantifier inside (or before) a capture can still
//               be retried with fewer repetitions if something later in the pattern
//               fails to match.
// opSaveEnd   : Record SP as the end of capture slot a. Same fall-through behavior.
// opNumber    : match against capture group %n
// opFrontier  : zero width assertion; matches when SP-1 is not in the class but
//               SP is (out of bounds counts as a NUL byte, as in Lua)
// opBalance   : %bxy; a is the begin byte, b is the end byte. Matches SP..e where
//               SP is x and e is the y that brings nesting back to zero, counting
//               every x as +1 and every y as -1 in between.

const (
	opChar op = iota
	opMatch
	opJmp
	opSplit
	opSaveStart
	opSaveEnd
	opNumber
	opFrontier
	opBalance
)

func (b *bytecode) String() string {
	switch b.op {
	case opChar:
		return fmt.Sprintf("CHAR %s", b.class)
	case opSaveStart:
		return fmt.Sprintf("SAVE_START %v", b.a)
	case opSaveEnd:
		return fmt.Sprintf("SAVE_END %v", b.a)
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
	case opBalance:
		return fmt.Sprintf("BALANCE %v %v", b.a, b.b)
	}
	return "UNKNOWN"
}

// Simple recursive virtual machine based on the
// "Regular Expression Matching: the Virtual Machine Approach" (https://swtch.com/~rsc/regexp/regexp2.html)
//
// src is matched as raw bytes rather than runes. Lua strings (and the patterns
// matched against them) are byte sequences, not necessarily valid UTF-8, so the
// indices in Match must line up with Go's native (byte based) string indexing.
//
// positionSlots has one entry per capture slot (slot 0 is the whole match, unused
// here) telling buildMatches whether that slot is a "()" position capture.
func eval(src []byte, instructions []bytecode, sp int, positionSlots []bool) (bool, int, []*Match, error) {
	caps := make([][2]int, len(positionSlots))
	for i := range caps {
		caps[i] = [2]int{-1, -1}
	}
	matched, _, nsp, caps, err := _eval(src, instructions, 0, sp, caps)
	if err != nil || !matched {
		return matched, nsp, nil, err
	}
	return true, nsp, buildMatches(src, caps, positionSlots), nil
}

func buildMatches(src []byte, caps [][2]int, positionSlots []bool) []*Match {
	matches := make([]*Match, 0, len(caps))
	for slot, c := range caps {
		if c[0] < 0 || c[1] < 0 {
			continue // capture never visited (shouldn't happen for a successful match, but be defensive)
		}
		if positionSlots[slot] {
			matches = append(matches, &Match{Start: c[0], End: c[0], IsPosition: true})
			continue
		}
		matches = append(matches, &Match{Start: c[0], End: c[1], Subs: string(src[c[0]:c[1]])})
	}
	return matches
}

func _eval(src []byte, instructions []bytecode, pc, sp int, caps [][2]int) (bool, int, int, [][2]int, error) {
	// Fresh backing array per frame: writes made while speculatively exploring a
	// branch that ultimately fails must never leak into the sibling branch tried
	// afterward (or into the caller's own state) once this frame's copy is dropped.
	caps = append([][2]int(nil), caps...)
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
			return matched, pc, sp, caps, nil
		case opJmp:
			pc += inst.a
		case opSplit:
			matched, npc, nsp, subCaps, err := _eval(src, instructions, pc+inst.a, sp, caps)
			if err != nil || matched {
				return matched, npc, nsp, subCaps, err
			}
			pc += inst.b
		case opSaveStart:
			caps[inst.a][0] = sp
			pc++
		case opSaveEnd:
			caps[inst.a][1] = sp
			pc++
		case opNumber:
			idx := inst.a
			if idx >= len(caps) || caps[idx][0] < 0 || caps[idx][1] < 0 {
				return false, pc, sp, nil, fmt.Errorf("invalid capture index %v", idx)
			}
			capture := src[caps[idx][0]:caps[idx][1]]
			for i, capt := range capture {
				if i+sp >= len(src) || capt != src[i+sp] {
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
		case opBalance:
			// Mirrors Lua's own matchbalance (lstrlib.c) exactly, including checking
			// the end byte before the begin byte: that ordering is what makes %bxy
			// with x==y degrade to "match up to the next occurrence" instead of
			// looping forever incrementing its own depth.
			beginCh, endCh := byte(inst.a), byte(inst.b)
			if sp >= len(src) || src[sp] != beginCh {
				return false, pc, sp, nil, nil
			}
			cont := 1
			i := sp
			for {
				i++
				if i >= len(src) {
					return false, pc, sp, nil, nil
				}
				if src[i] == endCh {
					cont--
					if cont == 0 {
						i++
						break
					}
				} else if src[i] == beginCh {
					cont++
				}
			}
			sp = i
			pc++
		default:
			return false, pc, sp, nil, errors.New("invalid operation happened while executing pattern")
		}
	}
}
