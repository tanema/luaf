package pattern

type (
	Pattern struct {
		src          string
		pattern      *seqPattern
		instructions []bytecode
	}
	PatternIterator struct {
		src    string
		pat    *Pattern
		offset int
	}
)

func Parse(src string) (*Pattern, error) {
	pat, err := parse(src)
	if err != nil {
		return nil, err
	}
	return &Pattern{
		src:          src,
		pattern:      pat,
		instructions: compile(pat),
	}, nil
}

func (p *Pattern) Iter(src string) PatternIterator {
	return PatternIterator{
		src:    src,
		pat:    p,
		offset: 0,
	}
}

func (p *Pattern) Find(src string, offset, limit int) ([]*Match, error) {
	allMatches := []*Match{}
	byteSrc := []byte(src)
	for offset <= len(byteSrc) {
		matched, newOffset, matches, err := p.Next(src, offset)
		if err != nil {
			return nil, err
		}
		if matched {
			allMatches = append(allMatches, matches...)
		}
		offset++
		if offset < newOffset {
			offset = newOffset
		}
		if len(matches) == limit || p.pattern.mustHead {
			break
		}
	}
	return allMatches, nil
}

func (p *Pattern) Next(src string, offset int) (bool, int, []*Match, error) {
	return eval([]byte(src), p.instructions, offset)
}

func (pi *PatternIterator) Next() ([]*Match, error) {
	for pi.offset <= len(pi.src) {
		matched, newOffset, matches, err := pi.pat.Next(pi.src, pi.offset)
		if err != nil {
			return nil, err
		}
		pi.offset++
		if pi.offset < newOffset {
			pi.offset = newOffset
		}
		if matched {
			return matches, nil
		}
	}
	return nil, nil
}
