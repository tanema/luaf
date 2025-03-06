package luaf

import (
	"fmt"
	"strings"
)

type Stack[T any] struct {
	data []*T
	top  int
}

func NewStack[T any](size int) Stack[T] {
	return Stack[T]{
		data: make([]*T, size),
		top:  0,
	}
}

func (s *Stack[T]) Top() *T {
	if s.top > 0 {
		return s.data[s.top-1]
	}
	return nil
}

func (s *Stack[T]) Push(vals ...*T) int {
	end := s.top + len(vals)
	if end >= len(s.data) {
		newSlice := make([]*T, end*2)
		copy(newSlice, s.data)
		s.data = newSlice
	}
	for _, val := range vals {
		s.data[s.top] = val
		s.top++
	}
	return s.top
}

func (s *Stack[T]) Pop() *T {
	if s.top <= 0 {
		return nil
	}
	s.top--
	value := s.data[s.top]
	s.data[s.top] = nil
	return value
}

func (s *Stack[T]) Len() int {
	return s.top
}

func printStackTrace(f *frame) string {
	parts := []string{}
	current := f
	for current != nil {
		parts = append(parts, fmt.Sprintf("\t%v", current))
		current = f.prev
	}
	return strings.Join(parts, "\n")
}

func unifyType(in any) any {
	switch val := in.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return float64(val)
	case float64:
		return float64(val)
	default:
		return in
	}
}

func ensureLenNil(values []Value, want int) []Value {
	if want <= 0 {
		return values
	} else if len(values) > want {
		values = values[:want:want]
	} else if len(values) < want {
		values = append(values, repeat[Value](&Nil{}, want-len(values))...)
	}
	return values
}

// ensureSize will ensure a slice has the correct length so that the index
// is not out of bounds but only resize to that exact amount and not more. This
// ensures that we can safely use an index if required
func ensureSize[T any](slice *[]T, index int) {
	sliceLen := len(*slice)
	if index < sliceLen {
		return
	}
	newSlice := make([]T, index+1)
	copy(newSlice, *slice)
	*slice = newSlice
}

// truncate will trim a slice down to an endpoint. This is good for discarding
// values that are out of scope
func truncate[T any](slice *[]T, index int) []T {
	if index >= len(*slice) || index < 0 {
		return []T{}
	} else if index < 0 {
		*slice = []T{}
		return []T{}
	}
	out := (*slice)[index:]
	*slice = (*slice)[:index:index]
	return out
}

// cutout will take out a chunk in the middle of a slice
func cutout[T any](slice *[]T, start, end int) {
	count := len(*slice)
	start = clamp(start, 0, count)
	end = clamp(end, 0, count)
	*slice = append((*slice)[:start], (*slice)[end:]...)
}

// repeat will generate a slice with a repeated value
func repeat[T any](x T, count int) []T {
	xs := make([]T, count)
	for i := 0; i < count; i++ {
		xs[i] = x
	}
	return xs
}

func reverse[T any](x []T) []T {
	for i, j := 0, len(x)-1; i < j; i, j = i+1, j-1 {
		x[i], x[j] = x[j], x[i]
	}
	return x
}

// search will find a value index in any slice with a comparison function passed
// this is good for slices of non-simple datatypes
func search[S ~[]E, E, T any](x S, target T, cmp func(E, T) bool) (int, bool) {
	for i := range x {
		if cmp(x[i], target) {
			return i, true
		}
	}
	return -1, false
}

func findLocal(lcl *local, name string) bool        { return name == lcl.name }
func findConst(k, name any) bool                    { return k == name }
func findUpindex(upindex UpIndex, name string) bool { return name == upindex.Name }
func findBroker(b *UpvalueBroker, idx uint64) bool  { return idx == b.index }

func b2U8(val bool) uint8 {
	if val {
		return 1
	}
	return 0
}

func clamp(f, low, high int) int {
	return min(max(f, low), high)
}

func substringIndex(val Value, strLen int) int64 {
	if i := toInt(val); i < 0 {
		return int64(strLen) + i
	} else if i == 0 {
		return 1
	} else {
		return i
	}
}
