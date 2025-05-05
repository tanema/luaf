package luaf

import (
	"context"
	"strings"
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
	for range b.N {
		execSnip(vm, src)
	}
}

func BenchmarkFibTailcall10(b *testing.B) {
	vm := NewVM(context.Background())
	src := `
local function fib(n, a, b)
	if n == 0 then
		return a
	elseif n == 1 then
		return b
	end
	return fib(n - 1, b, a + b)
end

fib(35, 0, 1)
`
	for range b.N {
		execSnip(vm, src)
	}
}

func execSnip(vm *VM, src string) {
	fn, err := Parse("<repl>", strings.NewReader(src), ModeText)
	if err != nil {
		panic(err)
	}
	_, err = vm.Eval(fn)
	if err != nil {
		panic(err)
	}
}
