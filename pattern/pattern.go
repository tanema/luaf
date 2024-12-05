package pattern

func Find(pattern, src string, offset, limit int) ([]*Match, error) {
	pat, err := parse(pattern)
	if err != nil {
		return nil, err
	}
	instructions := compile(pat)
	allMatches := []*Match{}
	byteSrc := []byte(src)
	for offset <= len(byteSrc) {
		matched, newOffset, matches, err := eval(byteSrc, instructions, offset)
		if err != nil {
			return nil, err
		} else if !matched {
			offset++
			continue
		}
		offset = newOffset
		allMatches = append(allMatches, matches...)
		if len(matches) == limit || pat.mustHead {
			break
		}
	}
	return allMatches, nil
}
