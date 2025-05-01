package luaf

func ensureLenNil(values []any, want int) []any {
	if want <= 0 {
		return values
	} else if len(values) > want {
		values = values[:want:want]
	} else if len(values) < want {
		for range want - len(values) {
			values = append(values, nil)
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

func substringIndex(val any, strLen int) int64 {
	i := toInt(val)
	if i < 0 {
		return int64(strLen) + i
	} else if i == 0 {
		return 1
	}
	return i
}
