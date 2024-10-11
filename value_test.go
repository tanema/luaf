package lauf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToValue(t *testing.T) {
	testcases := []struct {
		in  any
		out Value
	}{
		{in: int(11), out: &Integer{val: 11}},
		{in: int8(22), out: &Integer{val: 22}},
		{in: int16(33), out: &Integer{val: 33}},
		{in: int32(44), out: &Integer{val: 44}},
		{in: int64(55), out: &Integer{val: 55}},
		{in: uint(11), out: &Integer{val: 11}},
		{in: uint8(22), out: &Integer{val: 22}},
		{in: uint16(33), out: &Integer{val: 33}},
		{in: uint32(44), out: &Integer{val: 44}},
		{in: uint64(55), out: &Integer{val: 55}},
		{in: float32(44), out: &Float{val: 44}},
		{in: float64(55), out: &Float{val: 55}},
		{in: true, out: &Boolean{val: true}},
		{in: false, out: &Boolean{val: false}},
		{in: nil, out: &Nil{}},
		{in: "hello world", out: &String{val: "hello world"}},
		{in: &String{val: "hello world"}, out: &String{val: "hello world"}},
		{in: &VM{}, out: nil},
	}

	for _, tcase := range testcases {
		assert.Equal(t, tcase.out, ToValue(tcase.in))
	}
}
