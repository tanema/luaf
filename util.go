package luaf

import (
	"fmt"
	"slices"
	"strings"
)

type Stack[T any] []T

func (s *Stack[T]) Top() T {
	index := len(*s) - 1
	if index < 0 {
		var zero T
		return zero
	}
	return (*s)[index]
}

func (s *Stack[T]) Push(val ...T) int64 {
	addr := len(*s)
	(*s) = append(*s, val...)
	return int64(addr)
}

func (s *Stack[T]) Pop() T {
	var zero T
	index := len(*s) - 1
	if index < 0 {
		var zero T
		return zero
	}
	value := (*s)[index]
	(*s)[index] = zero
	*s = (*s)[:index]
	return value
}

func printStackTrace(stack Stack[*callInfo]) string {
	parts := []string{}
	for i := len(stack) - 1; i >= 0; i-- {
		parts = append(parts, fmt.Sprintf("\t%v", stack[i]))
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

// trimEndNil will check if there are nil values on the end of the slice, remove
// them and then resize the slice so that it is the exact size of the values
func trimEndNil(slice []Value) []Value {
	for i, val := range slice {
		if val == nil {
			return slices.Clip(slice[:i])
		}
	}
	return slice
}

func ensureLenNil(values []Value, want int) []Value {
	if len(values) > want {
		values = values[:want:want]
	} else if len(values) < want {
		values = append(values, repeat[Value](&Nil{}, want-len(values))...)
	}
	return values
}

// ensureSizeGrow will ensure a slice has the correct length so that the index
// is not out of bounds. It will also grow the slice in anticipation of more
// values in the future. This ensures that we can safely use an index if required
// and reduces the amount of times the slice needs to be resized
func ensureSizeGrow[T any](slice *[]T, index int) {
	sliceLen := len(*slice)
	if index < sliceLen {
		return
	}
	growthAmount := (index - (sliceLen - 1)) * 2
	newSlice := make([]T, sliceLen+growthAmount)
	copy(newSlice, *slice)
	*slice = newSlice
}

// ensureSizeGrow will ensure a slice has the correct length so that the index
// is not out of bound. This ensures that we can safely use an index if required
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
	end = clamp(end, 0, count+1)
	*slice = append((*slice)[:start-1], (*slice)[end-1:]...)
}

// repeat will generate a slice with a repeated value
func repeat[T any](x T, count int) []T {
	xs := make([]T, count)
	for i := 0; i < count; i++ {
		xs[i] = x
	}
	return xs
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

func findLocal(lcl *Local, name string) bool        { return name == lcl.name }
func findUpindex(upindex UpIndex, name string) bool { return name == upindex.name }
func findBroker(b *UpvalueBroker, idx int) bool     { return idx == b.index }

func b2U8(val bool) uint8 {
	if val {
		return 1
	}
	return 0
}

func clamp(f, low, high int) int {
	if f < low {
		return low
	} else if f > high {
		return high
	}
	return f
}
