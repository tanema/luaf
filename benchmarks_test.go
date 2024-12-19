package luaf

import (
	"bytes"
	"context"
	"testing"
)

func BenchmarkFib10(b *testing.B) {
	vm := NewVM(context.Background())
	src := `
local function fib(n)
    if n < 2 then return n end
    return fib(n - 2) + fib(n - 1)
end

fib(10)`
	for n := 0; n < b.N; n++ {
		execSnip(vm, src)
	}
}

func execSnip(vm *VM, src string) {
	fn, err := Parse("<repl>", bytes.NewBufferString(src))
	if err != nil {
		panic(err)
	}
	_, err = vm.Eval(fn)
	if err != nil {
		panic(err)
	}
}
