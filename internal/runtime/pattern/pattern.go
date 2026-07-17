// Package pattern is the package that implements lua patterns. Patterns in Lua
// are described by regular strings, which are interpreted as
//
// patterns by the pattern-matching functions string.find, string.gmatch, string.gsub,
// and string.match. This section describes the syntax and the meaning (that is,
// what they match) of these strings.
//
// Character Class:
// A character class is used to represent a set of characters. The following
// combinations are allowed in describing a character class:
//
//   - x: (where x is not one of the magic characters ^$()%.[]*+-?) represents the character x itself.
//   - .: (a dot) represents all characters.
//   - %a: represents all letters.
//   - %c: represents all control characters.
//   - %d: represents all digits.
//   - %g: represents all printable characters except space.
//   - %l: represents all lowercase letters.
//   - %p: represents all punctuation characters.
//   - %s: represents all space characters.
//   - %u: represents all uppercase letters.
//   - %w: represents all alphanumeric characters.
//   - %x: represents all hexadecimal digits.
//   - %x: (where x is any non-alphanumeric character) represents the character x.
//   - [set]: represents the class which is the union of all characters in set.
//
// You can put a closing square bracket in a set by positioning it as the first
// character in the set. You can put a hyphen in a set by positioning it as the
// first or the last character in the set. (You can also use an escape for both cases.)
// For all classes represented by single letters (%a, %c, etc.), the corresponding
// uppercase letter represents the complement of the class. For instance, %S
// represents all non-space characters.
//
// The definitions of letter, space, and other character groups depend on the
// current locale. In particular, the class [a-z] may not be equivalent to %l.
//
// Pattern Item:
//
// A pattern item can be
//
//   - a single character class, which matches any single character in the class;
//   - a single character class followed by '*', which matches zero or more repetitions of characters in the class.
//   - a single character class followed by '+', which matches one or more repetitions of characters in the class.
//   - a single character class followed by '-', which also matches zero or more repetitions of characters in the class
//   - a single character class followed by '?', which matches zero or one occurrence of a character in the class.
//   - %n, for n between 1 and 9; such item matches a substring equal to the n-th captured string (see below);
//   - %bxy, where x and y are two distinct characters;
//   - %f[set], a frontier pattern;
//
// Pattern:
//
// A pattern is a sequence of pattern items. A caret '^' at the beginning of a
// pattern anchors the match at the beginning of the subject string. A '$' at the
// end of a pattern anchors the match at the end of the subject string. At other
// positions, '^' and '$' have no special meaning and represent themselves.
//
// Captures:
//
// A pattern can contain sub-patterns enclosed in parentheses; they describe captures.
// When a match succeeds, the substrings of the subject string that match captures
// are stored (captured) for future use. Captures are numbered according to their
// left parentheses. For instance, in the pattern "(a*(.)%w(%s*))", the part of
// the string matching "a*(.)%w(%s*)" is stored as the first capture (and therefore
// has number 1); the character matching "." is captured with number 2, and the
// part matching "%s*" has number 3.
package pattern

import "fmt"

type (
	// Pattern is a parsed pattern from a string into a pattern bytecode that can be reused.
	Pattern struct {
		src           string
		pattern       *seqPattern
		instructions  []bytecode
		positionSlots []bool
	}
	// Iterator allows for iteraterating on a pattern for each match in a string.
	Iterator struct {
		pat          *Pattern
		src          string
		offset       int
		lastMatchEnd int // end offset of the last accepted match; -1 if none yet
	}
)

// SpecialChars are characters that are used in patterns.
const SpecialChars = "^$*+?.([%-"

// Parse will parse a string pattern into a bytecode operations that can be matched
// on a string.
func Parse(src string) (*Pattern, error) {
	pat, err := parse(src)
	if err != nil {
		return nil, err
	}
	instructions, positionSlots := compile(pat, pat.numCaps)
	return &Pattern{
		src:           src,
		pattern:       pat,
		instructions:  instructions,
		positionSlots: positionSlots,
	}, nil
}

// Find is an easy method to find the first match with a pattern.
func Find(pat, src string) ([]*Match, error) {
	parsedPattern, err := Parse(pat)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	matches, err := parsedPattern.Find(src, 1)
	if err != nil {
		return nil, fmt.Errorf("bad pattern: %w", err)
	}
	return matches, nil
}

// Iter creates a new iterator on a string, starting at the given byte offset.
func (p *Pattern) Iter(src string, offset int) Iterator {
	return Iterator{
		src:          src,
		pat:          p,
		offset:       offset,
		lastMatchEnd: -1,
	}
}

// Find finds a match with a pattern in a string with a limit. If the limit is negative
// all matches are found.
func (p *Pattern) Find(src string, limit int) ([]*Match, error) {
	offset := 0
	found := 0
	lastMatchEnd := -1
	allMatches := []*Match{}
	byteSrc := []byte(src)
	for offset <= len(byteSrc) {
		matched, newOffset, matches, err := p.Next(src, offset)
		if err != nil {
			return nil, err
		}
		// An empty match landing exactly where the previous accepted match ended is
		// not a new match: without this it double-counts (e.g. " *" over "a b" would
		// match, then immediately re-match empty at the same spot the space match
		// just ended). Reject it and fall through to the ordinary one-byte advance.
		if matched && newOffset == lastMatchEnd {
			matched = false
		}
		if matched {
			allMatches = append(allMatches, matches...)
			found++
			lastMatchEnd = newOffset
		}
		offset++
		if offset < newOffset {
			offset = newOffset
		}
		if found == limit || p.pattern.mustHead {
			break
		}
	}
	return allMatches, nil
}

// Next will return the next match if there is one. It will return false if no
// match was found.
func (p *Pattern) Next(src string, offset int) (bool, int, []*Match, error) {
	return eval([]byte(src), p.instructions, offset, p.positionSlots)
}

// Next will return the next match in the iterator. It will return nil otherwise.
func (pi *Iterator) Next() ([]*Match, error) {
	for pi.offset <= len(pi.src) {
		matched, newOffset, matches, err := pi.pat.Next(pi.src, pi.offset)
		if err != nil {
			return nil, err
		}
		// See the identical check in Pattern.Find: don't accept an empty match that
		// ends exactly where the last accepted match ended.
		if matched && newOffset == pi.lastMatchEnd {
			matched = false
		}
		pi.offset++
		if pi.offset < newOffset {
			pi.offset = newOffset
		}
		if matched {
			pi.lastMatchEnd = newOffset
			return matches, nil
		}
	}
	return nil, nil
}
