package pattern

func compile(p any) []bytecode {
	instructions := append([]bytecode{{op: opSave, a: 1}}, compilePattern(p)...)
	if p.(*seqPattern).mustTail {
		instructions = append(instructions, bytecode{op: opMatch, a: 1})
	} else {
		instructions = append(instructions, bytecode{op: opMatch})
	}
	return instructions
}

func compilePattern(p any) []bytecode {
	insts := []bytecode{}
	switch pat := p.(type) {
	case *singlePattern:
		insts = append(insts, bytecode{op: opChar, class: pat.class})
	case *seqPattern:
		for _, cp := range pat.patterns {
			insts = append(insts, compilePattern(cp)...)
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
		insts = append(insts,
			bytecode{op: opChar, class: &charClass{ch: pat.begin}}, // check begin
			bytecode{op: opSplit, a: 1, b: 3},                      // do next char, if no match jump after jmp
			bytecode{op: opChar, class: &dotClass{}},               // .*
			bytecode{op: opJmp, a: -2},                             // jmp back to split
			bytecode{op: opChar, class: &charClass{ch: pat.end}},   // check end
		)
	case *capPattern:
		insts = append([]bytecode{{op: opSave}}, append(compilePattern(pat.pattern), bytecode{op: opMatch})...)
	case *numberPattern:
		insts = append(insts, bytecode{op: opNumber, a: int(pat.n) - 1})
	}
	return insts
}
