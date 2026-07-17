package pattern

// compile turns a parsed pattern into bytecode. Slot 0 is reserved for the whole
// match; capPattern.slot (assigned during parsing, in the order '(' appears)
// gives every user capture its own slot. positionSlots[slot] tells the caller
// whether that slot is a "()" position capture rather than a text capture.
func compile(p any, numCaps int) ([]bytecode, []bool) {
	positionSlots := make([]bool, numCaps+1)
	instructions := append([]bytecode{{op: opSaveStart, a: 0}}, compilePattern(p, positionSlots)...)
	seq, _ := p.(*seqPattern)
	endMatch := bytecode{op: opMatch}
	if seq.mustTail {
		endMatch.a = 1
	}
	instructions = append(instructions, bytecode{op: opSaveEnd, a: 0}, endMatch)
	return instructions, positionSlots
}

func compilePattern(p any, positionSlots []bool) []bytecode {
	insts := []bytecode{}
	switch pat := p.(type) {
	case *singlePattern:
		insts = append(insts, bytecode{op: opChar, class: pat.class})
	case *seqPattern:
		for _, cp := range pat.patterns {
			insts = append(insts, compilePattern(cp, positionSlots)...)
		}
	case *repeatPattern:
		switch pat.kind {
		case '*':
			insts = append(insts,
				bytecode{op: opSplit, a: 1, b: 3}, // do next char, if no match jump after jmp
				bytecode{op: opChar, class: pat.class},
				bytecode{op: opJmp, a: -2}, // jmp back to split
			)
		case '+':
			insts = append(insts,
				bytecode{op: opChar, class: pat.class},
				bytecode{op: opSplit, a: -1, b: 1}, // if successful go back to +, else continue
			)
		case '-':
			insts = append(insts,
				bytecode{op: opSplit, a: 3, b: 1},
				bytecode{op: opChar, class: pat.class},
				bytecode{op: opJmp, a: -2},
			)
		case '?':
			insts = append(insts,
				bytecode{op: opSplit, a: 1, b: 2},
				bytecode{op: opChar, class: pat.class},
			)
		}
	case *bracePattern:
		// %bxy matches balanced, nestable x/y pairs (e.g. parens), not just "x
		// followed eventually by y" - it has to track nesting depth, which a
		// backtracking char-by-char split/jmp sequence can't express. opBalance
		// does the deterministic depth-counting scan directly.
		insts = append(insts, bytecode{op: opBalance, a: int(pat.begin), b: int(pat.end)})
	case *capPattern:
		positionSlots[pat.slot] = pat.isPosition
		insts = append(insts, bytecode{op: opSaveStart, a: pat.slot})
		insts = append(insts, compilePattern(pat.pattern, positionSlots)...)
		insts = append(insts, bytecode{op: opSaveEnd, a: pat.slot})
	case *numberPattern:
		insts = append(insts, bytecode{op: opNumber, a: int(pat.n)})
	case *frontierPattern:
		insts = append(insts, bytecode{op: opFrontier, class: pat.set})
	}
	return insts
}
