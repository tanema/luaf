package luaf

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
	out := (*slice)[index:]
	*slice = (*slice)[:index:index]
	return out
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

func findLocal(lcl string, name string) bool        { return name == lcl }
func findUpindex(upindex UpIndex, name string) bool { return name == upindex.name }
func findBroker(b *Broker, idx int) bool            { return idx == b.index }

func b2U8(val bool) uint8 {
	if val {
		return 1
	}
	return 0
}
