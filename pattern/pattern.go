package pattern

func Find(pattern, src string, offset, limit int) ([]*Match, error) {
	pat, err := parse(pattern)
	if err != nil {
		return nil, err
	}
	instructions := compile(pat)
	matches := []*Match{}
	byteSrc := []byte(src)
	for sp := offset; sp <= len(byteSrc); {
		ms, err := eval(byteSrc, instructions, 0, sp)
		if err != nil {
			return nil, err
		}
		sp++
		if ms != nil {
			if sp < ms.End {
				sp = ms.End
			}
			matches = append(matches, ms)
		}
		if len(matches) == limit || pat.mustHead {
			break
		}
	}
	return matches, nil
}
