package pattern

type Pattern struct {
	src          string
	pattern      *seqPattern
	instructions []bytecode
}

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

func (p *Pattern) FindAll(src string) ([]*Match, error) {
	return p.Find(src, 0, -1)
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
