package lsp

import "fmt"

type RPCError struct {
	requestID *int
	code      ErrorCode
	message   string
	data      any
}

func rpcErr(id *int, code ErrorCode, message string, data any) *RPCError {
	return &RPCError{
		requestID: id,
		code:      code,
		message:   message,
		data:      data,
	}
}

func (err *RPCError) Error() string {
	return fmt.Sprintf("RPC Error id: %d code: %d, err: %s, data: %v", err.requestID, err.code, err.message, err.data)
}

func (err *RPCError) toResponse() *ResponseErrorMessage {
	return &ResponseErrorMessage{
		JSONRPC: "2.0",
		ID:      err.requestID,
		Error: ResponseError{
			Code:    err.code,
			Message: err.message,
			Data:    err.data,
		},
	}
}
