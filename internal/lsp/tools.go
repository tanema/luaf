package lsp

import (
	"context"
	"encoding/json"
)

func ToPtr[T any](x T) *T {
	return &x
}

func FromPtr[T any](x *T) T {
	var zero T
	return FromPtrDefault(x, zero)
}

func FromPtrDefault[T any](x *T, def T) T {
	if x == nil {
		return def
	}
	return *x
}

func callHandler[T any](ctx context.Context, id *int, payload []byte, fn func(ctx context.Context, id *int, params T) error) error {
	req := struct {
		Params T `json:"params"`
	}{}
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	return fn(ctx, id, req.Params)
}
