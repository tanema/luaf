package pattern

import (
	"bytes"
	"fmt"
	"unicode"
)

const (
	LUA_MAXCAPTURES = 32
	CAP_UNFINISHED  = -1
)

type (
	capture struct {
		src []byte
		len int
	}
	matchState struct {
		src     []byte
		level   int
		capture []capture
	}
)

// Returns whether or not a byte matches a specified character class, which may
// simply be a non-special character itself.
func matchClass(c byte, cl byte) (res bool) {
	switch unicode.ToLower(rune(cl)) {
	case 'a':
		res = unicode.IsLetter(rune(c))
	case 'c':
		res = unicode.IsControl(rune(c))
	case 'd':
		res = unicode.IsDigit(rune(c))
	case 'l':
		res = unicode.IsLower(rune(c))
	case 'p':
		res = unicode.IsPunct(rune(c))
	case 's':
		res = unicode.IsSpace(rune(c))
	case 'u':
		res = unicode.IsUpper(rune(c))
	case 'w':
		res = unicode.IsLetter(rune(c)) || unicode.IsNumber(rune(c))
	case 'x':
		res = unicode.IsDigit(rune(c)) || unicode.IsLetter(rune(c))
	case 'z':
		res = (c == 0)
	default:
		return cl == c
	}
	if unicode.IsLower(rune(cl)) {
		return res
	}
	return !res // handle upper-case reverse classes
}

// matchBracketClass returns whether or not a given character matches the character class
// specified in the pattern.
func matchBracketClass(c byte, p []byte, ec []byte) bool {
	var sig bool = true
	if p[1] == '^' {
		sig = false
		p = p[1:]
	}
	for p = p[1:]; len(p) > len(ec); p = p[1:] {
		if p[0] == '%' {
			p = p[1:]
			if matchClass(c, p[0]) {
				return sig
			}
		} else if p[1] == '-' && (len(p)-2 > len(ec)) {
			if p[0] <= c && c <= p[2] {
				return sig
			}
		} else if p[0] == c {
			return sig
		}
	}

	return !sig
}

// singleMatch returns whether or not a single character matches the pattern currently
// being examined. It needs to 'lookahead' in order to accomplish this, for
// example when the pattern is a character class like %l. This is the purpose
// of the argument ep.
func singleMatch(c byte, p []byte, ep []byte) bool {
	switch p[0] {
	case '.':
		return true
	case '%':
		return matchClass(c, p[1])
	case '[':
		ep = p[len(p)-len(ep)-1:]
		return matchBracketClass(c, p, ep)
	default:
		return p[0] == c
	}
}

// Returns the portion of the source string that matches the balance pattern
// specified, where b is the start and e is the end of the balance pattern.
func matchBalance(s, p []byte) []byte {
	if len(p) <= 1 {
		// error: unbalanced pattern
		return nil
	}
	if s[0] != p[0] {
		return nil
	} else {
		var b byte = p[0]
		var e byte = p[1]
		var cont int = 1

		// ms.src_end in the original C source is a pointer to the end of the
		// source string (whatever that means specifically). This loop wants to
		// ensure that s remains less than this pointer. Since we're not
		// dealing with pointers, we should be able to just run the loop until
		// s runs out.
		for s = s[1:]; len(s) > 0; s = s[1:] {
			if s[0] == e {
				cont = cont - 1
				if cont == 0 {
					return s[1:]
				}
			} else if s[0] == b {
				cont = cont + 1
			}
		}
	}

	// error: strings ends out of balance
	return nil
}

// Return the maximum portion of the source string that matches the given
// pattern (equates to the '+' or '*' operator)
func maxExpand(ms *matchState, s, p, ep []byte) []byte {
	// Run through the string to find the maximum number of matches that are
	// possible for the pattern item.
	var i int // count maximum expand for item
	for i = 0; i < len(s) && singleMatch(s[i], p, ep); i++ {
	}

	// Try to match with maximum reptitions
	for i >= 0 {
		res := match(ms, s[i:], ep[1:])
		if res != nil {
			return res
		} else {
			// Reduce 1 repetition and try again
			i--
		}
	}
	return nil
}

// Returns the minimum portion of the source string that matches the given
// pattern (equates to the '-' operator)
func minExpand(ms *matchState, s, p, ep []byte) []byte {
	for {
		res := match(ms, s, ep[1:])
		if res != nil {
			return res
		} else if len(s) > 0 && singleMatch(s[0], p, ep) {
			// try with one more repetition
			s = s[1:]
		} else {
			return nil
		}
	}
}

// Checks if a capture exists with the given capture index. Rather than
// providing an error, since we're not sure how we'd do that right now, we
// return -1 and handle error checking outside this routine.
func checkCapture(ms *matchState, l int) int {
	l = l - '1'
	if l < 0 || l >= ms.level || ms.capture[l].len == CAP_UNFINISHED {
		// error: invalid capture index
		return -1
	}
	return l
}

// Returns the first level that contains an unclosed capture, or -1 if there is
// no such capture level.
func captureToClose(ms *matchState) int {
	for level := ms.level - 1; level >= 0; level = level - 1 {
		if ms.capture[level].len == CAP_UNFINISHED {
			return level
		}
	}
	panic("NO SOUP FOR YOU")
}

// Finds the end of a character class [] and return that part of the pattern
func classEnd(p []byte) []byte {
	var ch byte = p[0]
	p = p[1:]

	switch ch {
	case '%':
		if len(p) == 0 {
			// error: malformed pattern, ends with '%'
			return nil
		}
		return p[1:]
	case '[':
		if p[0] == '^' {
			p = p[1:]
		}
		// look for a ']'
		for {
			if len(p) == 0 {
				// error: malformed pattern (missing ']')
				return nil
			}
			pch := p[0]
			p = p[1:]
			if pch == '%' && len(p) > 0 {
				// skip escapes (e.g. %])
				p = p[1:]
			}
			if p[0] == ']' {
				break
			}
		}
		return p[1:]
	default:
		return p
	}
}

func startCapture(ms *matchState, s, p []byte, what int) []byte {
	var res []byte
	var level int = ms.level

	if level >= LUA_MAXCAPTURES {
		// error: too many captures
		return nil
	}
	ms.capture[level].src = s
	ms.capture[level].len = what
	ms.level = level + 1
	if res = match(ms, s, p); res == nil {
		ms.level--
	}
	return res
}

func endCapture(ms *matchState, s, p []byte) []byte {
	var l int = captureToClose(ms)
	if l == -1 {
		return nil
	}
	var res []byte
	ms.capture[l].len = len(ms.capture[l].src) - len(s)
	if res = match(ms, s, p); res == nil {
		ms.capture[l].len = CAP_UNFINISHED
	}
	return res
}

func matchCapture(ms *matchState, s []byte, l int) []byte {
	var clen int
	l = checkCapture(ms, l)
	if l == -1 {
		return nil
	}
	clen = ms.capture[l].len
	if len(s)-clen >= 0 && bytes.Equal(ms.capture[l].src[0:clen], s[0:clen]) {
		return s[clen:]
	}
	return nil
}

func match(ms *matchState, s, p []byte) []byte {
init:
	if len(p) == 0 {
		return s
	}
	var ep []byte
	var m bool
	switch p[0] {
	case '(':
		if p[1] == ')' { // position capture
			return nil // TODO: We don't support these
		} else {
			return startCapture(ms, s, p[1:], CAP_UNFINISHED)
		}
	case ')':
		return endCapture(ms, s, p[1:])
	case '%':
		switch p[1] {
		case 'b':
			s = matchBalance(s, p[2:])
			if s == nil {
				return nil
			}
		default: // TODO: Support the frontier pattern
			if unicode.IsDigit(rune(p[1])) { // capture result (%0-%9)
				s = matchCapture(ms, s, int(p[1]))
				if s == nil {
					return nil
				}
				p = p[2:]
				goto init
			}
			goto dflt
		}
	case '$': // check to ensure that the '$' is the last character in the pattern
		if len(p) == 1 {
			if len(s) == 0 {
				return s
			} else {
				return nil
			}
		} else {
			goto dflt
		}
	default:
		goto dflt
	}

	goto skipdflt

dflt: // it is a pattern item
	ep = classEnd(p) // points to what is next
	m = len(s) > 0 && singleMatch(s[0], p, ep)

	// Handle the case where ep has run out so we can't index it
	if len(ep) == 0 {
		if !m {
			return nil
		} else {
			s = s[1:]
			p = ep
			goto init
		}
	}

	switch ep[0] {
	case '?':
		if len(s) == 0 {
			return []byte{}
		}
		var res []byte = match(ms, s[1:], ep[1:])
		if m && res != nil {
			return res
		}
		p = ep[1:]
		goto init
	case '*':
		return maxExpand(ms, s, p, ep)
	case '+':
		if m {
			return maxExpand(ms, s[1:], p, ep)
		} else {
			return nil
		}
	case '-':
		return minExpand(ms, s, p, ep)
	default:
		if !m {
			return nil
		}
		s = s[1:]
		p = ep
		goto init
	}

skipdflt:

	return nil
}

func getOneCapture(ms *matchState, i int, s, e []byte) ([]byte, error) {
	if i >= ms.level {
		if i == 0 {
			// return whole match
			return s[0 : len(s)-len(e)], nil
		} else {
			return nil, fmt.Errorf("invalid capture index %v", i)
		}
	} else {
		var l int = ms.capture[i].len
		if l == CAP_UNFINISHED {
			return nil, fmt.Errorf("unfinished capture")
		} else {
			return ms.capture[i].src[0:l], nil
		}
	}
}

// Returns the index in 's1' where the 's2' can be found, or -1
func lmemfind(s1 []byte, s2 []byte) int {
	l1, l2 := len(s1), len(s2)
	if l2 == 0 {
		return 0
	} else if l2 > l1 {
		return -1
	} else {
		init := bytes.IndexByte(s1, s2[0])
		end := init + l2
		for end <= l1 && init != -1 {
			init++ // 1st char is already checked by IndexBytes
			if bytes.Equal(s1[init-1:end], s2) {
				return init - 1
			} else { // find the next 'init' and try again
				next := bytes.IndexByte(s1[init:], s2[0])
				if next == -1 {
					return -1
				} else {
					init = init + next
					end = init + l2
				}
			}
		}
	}
	return -1
}

func addS(ms *matchState, b *bytes.Buffer, s, e []byte, news []byte) error {
	l := len(news)
	for i := 0; i < l; i++ {
		if news[i] != '%' {
			b.WriteByte(news[i])
		} else {
			i++ // skip ESC (%)
			if !unicode.IsDigit(rune(news[i])) {
				b.WriteByte(news[i])
			} else if news[i] == '0' {
				b.Write(s[0 : len(s)-len(e)])
			} else {
				cidx := int(news[i] - '1')
				capt, err := getOneCapture(ms, cidx, s, e)
				if err != nil {
					return err
				}
				b.Write(capt)
			}
		}
	}
	return nil
}

// Looks for the first match of pattern p in the string s. If it finds one,
// then match returns true and the captures from the pattern; otherwise it
// returns false, nil.  If pattern specifies no captures, then the whole match
// is returned.
func Match(s, p string) (bool, []string) {
	sb, pb := []byte(s), []byte(p)
	succ, _, _, caps := FindBytes(sb, pb, false)

	scaps := make([]string, len(caps))
	for idx, str := range caps {
		scaps[idx] = string(str)
	}

	return succ, scaps[0:len(caps)]
}

// Same as the Match function, however operates directly on byte arrays rather
// than strings. This package operates natively in bytes, so this function is
// called by Match to perform it's work.
func MatchBytes(s, p []byte) (bool, [][]byte) {
	succ, _, _, caps := FindBytes(s, p, false)
	return succ, caps
}

// Returns a channel that can be used to iterate over all the matches of
// pattern p in string s. The single value sent down this channel is an
// array of the captures from the match.
func Gmatch(s, p string) chan []string {
	out := make(chan []string)
	start := 0
	go func() {
		for {
			succ, _, e, caps := Find(s[start:], p, false)
			if !succ {
				close(out)
				return
			} else {
				out <- caps
				start = e + start
			}
		}
	}()

	return out
}

// Same as the Gmatch function, however operates directly on byte arrays rather
// than strings. This package operates natively in bytes, so this function is
// called by Gmatch to perform it's work.
func GmatchBytes(s, p []byte) chan [][]byte {
	out := make(chan [][]byte)
	start := 0
	go func() {
		for {
			succ, _, e, caps := FindBytes(s[start:], p, false)
			if succ {
				out <- caps
				start = e
			} else {
				close(out)
				return
			}
		}
	}()

	return out
}

// Looks for the first match of pattern p in the string s. If it finds a match,
// then find returns the indices of s where this occurrence starts and ends;
// otherwise, it returns nil. If the pattern has captures, they are returned in
// an array. If the argument 'plain' is set to 'true', then this function
// performs a plain 'find substring' operation with no characters in the
// pattern being considered magic.
//
// Note that the indices returned from this function will NOT match the
// versions returned by the equivalent Lua string and pattern due to the
// differences in slice semantics and array indexing.
//
// You can rely on the fact that s[startIdx:endIdx] will be the entire portion
// of the string that matched the pattern.
func Find(s, p string, plain bool) (bool, int, int, []string) {
	sb, pb := []byte(s), []byte(p)
	succ, start, end, caps := FindBytes(sb, pb, plain)

	scaps := make([]string, len(caps))
	for idx, str := range caps {
		scaps[idx] = string(str)
	}

	return succ, start, end, scaps[0:len(caps)]
}

// Same as the Find function, however operates directly on byte arrays rather
// than strings. This package operates natively in bytes, so this function is
// called by Find to perform it's work.
func FindBytes(s, p []byte, plain bool) (bool, int, int, [][]byte) {
	if plain || !bytes.ContainsAny(p, "^$*+?.([%-") {
		if index := lmemfind(s, p); index != -1 {
			return true, index, index + len(p), nil
		} else {
			return false, -1, -1, nil
		}
	}

	// Perform a find and capture, looping to potentially find a match later in
	// the string
	var anchor bool = false
	if p[0] == '^' {
		p = p[1:]
		anchor = true
	}
	ms := new(matchState)
	ms.capture = make([]capture, LUA_MAXCAPTURES)

	var init int = 0
	for {
		res := match(ms, s[init:], p)

		if res != nil {
			// Determine the start and end indices of the match
			var start int = init
			var end int = len(s) - len(res)

			// Fetch the captures
			captures := new([LUA_MAXCAPTURES][]byte)

			var i int
			var nlevels int
			if ms.level == 0 && len(s) > 0 {
				nlevels = 1
			} else {
				nlevels = ms.level
			}

			for i = 0; i < nlevels; i++ {
				capt, err := getOneCapture(ms, i, s, res)
				if err != nil {
					return false, -1, -1, nil
				}
				captures[i] = capt
			}

			return true, start, end, captures[0:nlevels]
		} else if len(s)-init == 0 || anchor {
			break
		}

		init = init + 1
	}
	// No match found
	return false, -1, -1, nil
}

func Replace(src, patt, repl string, max int) (string, int) {
	res, n := ReplaceBytes([]byte(src), []byte(patt), []byte(repl), max)
	return string(res), n
}

// Same as the Replace function, however operates directly on byte arrays
// rather than strings. This package operates natively in bytes, so this
// function is called by Replace to perform it's work.
func ReplaceBytes(src, patt, repl []byte, max int) ([]byte, int) {
	anchor := false
	if patt[0] == '^' {
		anchor = true
		patt = patt[1:]
	}

	var n int = 0
	var b bytes.Buffer
	ms := new(matchState)
	ms.src = src
	ms.capture = make([]capture, LUA_MAXCAPTURES)

	for n < max || max == -1 {
		ms.level = 0
		e := match(ms, src, patt)
		if e != nil {
			n++
			if err := addS(ms, &b, src, e, repl); err != nil {
				panic(err)
			} // Use addS directly here
		}
		if e != nil && len(src) > 0 { // Non empty match
			src = e // skip it
		} else if len(src) > 0 {
			b.WriteByte(src[0])
			src = src[1:]
		} else {
			break
		}

		if anchor {
			break
		}
	}
	b.Write(src[0:])
	return b.Bytes(), n
}
