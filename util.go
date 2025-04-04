package luaf

import (
	"fmt"
	"strings"
)

type stack[T any] struct {
	data []*T
	top  int
}

func newStack[T any](size int) stack[T] {
	return stack[T]{
		data: make([]*T, size),
		top:  0,
	}
}

func (s *stack[T]) Top() *T {
	if s.top > 0 {
		return s.data[s.top-1]
	}
	return nil
}

func (s *stack[T]) Push(vals ...*T) int {
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

func (s *stack[T]) Pop() *T {
	if s.top <= 0 {
		return nil
	}
	s.top--
	value := s.data[s.top]
	s.data[s.top] = nil
	return value
}

func (s *stack[T]) Len() int {
	return s.top
}

func printStackTrace(callstack stack[callInfo]) string {
	parts := []string{}
	for i := range callstack.top {
		parts = append(parts, fmt.Sprintf("\t%v", callstack.data[i]))
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
		for range want - len(values) {
			values = append(values, &Nil{})
		}
	}
	return values
}

// ensures that we can safely use an index if required.
func ensureSize[T any](slice *[]T, index int) {
	sliceLen := len(*slice)
	if index < sliceLen {
		return
	}
	newSlice := make([]T, index+1)
	copy(newSlice, *slice)
	*slice = newSlice
}

// this is good for slices of non-simple datatypes.
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
func findBroker(b *upvalueBroker, idx uint64) bool  { return idx == b.index }

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
	i := toInt(val)
	if i < 0 {
		return int64(strLen) + i
	} else if i == 0 {
		return 1
	}
	return i
}
