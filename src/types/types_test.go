package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeCheck(t *testing.T) {
	cases := []struct {
		a, b  Definition
		match bool
	}{
		{Any, Nil, true},
		{Any, Int, true},
		{Any, Float, true},
		{Any, Bool, true},
		{Any, String, true},
		{Nil, Nil, true},
		{Nil, String, false},
		{Number, Int, true},
		{Number, Float, true},
		{Number, String, false},
		{Int, Int, true},
		{Float, Float, true},
		{Float, Number, false},
		{Bool, Bool, true},
		{String, String, true},
		{&Union{Defn: []Definition{String, Nil}}, String, true},
		{&Union{Defn: []Definition{String, Nil}}, Nil, true},
		{&Union{Defn: []Definition{String, Number}}, String, true},
		{&Union{Defn: []Definition{String, Number}}, Int, true},
		{&Union{Defn: []Definition{String, Number}}, Bool, false},
		{&Function{Params: []NamedPair{{"name", String}}, Return: []Definition{Any}}, &Function{Params: []NamedPair{{"name", String}}, Return: []Definition{Any}}, true},
		{&Function{Params: []NamedPair{}, Return: []Definition{Any}}, &Function{Params: []NamedPair{}, Return: []Definition{Any}}, true},
		{&Function{Params: []NamedPair{}, Return: []Definition{}}, &Function{Params: []NamedPair{}, Return: []Definition{}}, true},
		{&Function{Params: []NamedPair{{"name", String}}, Return: []Definition{}}, &Function{Params: []NamedPair{}, Return: []Definition{}}, false},
		{&Function{Params: []NamedPair{}, Return: []Definition{}}, &Function{Params: []NamedPair{{"name", String}}, Return: []Definition{}}, false},
		{&Function{Params: []NamedPair{}, Return: []Definition{}}, &Function{Params: []NamedPair{}, Return: []Definition{Int}}, false},
		{&Function{Params: []NamedPair{}, Return: []Definition{Int}}, &Function{Params: []NamedPair{}, Return: []Definition{}}, false},
	}

	for i, tc := range cases {
		res := tc.a.Check(tc.b)
		assert.Equal(t, tc.match, res, "[%v] %s does not match %s", i, tc.a, tc.b)
	}
}

func TestTypeString(t *testing.T) {
	cases := []struct {
		defn     Definition
		expected string
	}{
		{Any, NameAny},
		{Nil, NameNil},
		{Number, "{int | float}"},
		{Int, NameInt},
		{Float, NameFloat},
		{Bool, NameBool},
		{String, NameString},
		{&Union{Defn: []Definition{String, Nil}}, "{string | nil}"},
		{&Intersection{Defn: []Definition{String, Number}}, "{string & {int | float}}"},
		{&Function{Params: []NamedPair{{"name", String}}, Return: []Definition{Any}}, "function(name: string): any"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, tc.defn.String())
	}
}
