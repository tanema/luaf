package luaf

// this is good for slices of non-simple datatypes.
func search[S ~[]E, E, T any](x S, target T, cmp func(E, T) bool) (int, bool) {
	for i := range x {
		if cmp(x[i], target) {
			return i, true
		}
	}
	return -1, false
}

func findLocal(lcl *local, name string) bool       { return name == lcl.name }
func findConst(k, name any) bool                   { return k == name }
func findUpindex(ui upindex, name string) bool     { return name == ui.Name }
func findBroker(b *upvalueBroker, idx uint64) bool { return idx == b.index }
