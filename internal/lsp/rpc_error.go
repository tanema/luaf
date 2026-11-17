package lsp

import (
	"fmt"
	"strconv"
)

// RPCError is an error a handler returns to send a specific JSON-RPC error code to the client.
type RPCError struct {
	requestID *int
	code      ErrorCode
	message   string
	data      any
}

func rpcErr(id *int, code ErrorCode, message string, data any) *RPCError { //nolint:unparam
	return &RPCError{
		requestID: id,
		code:      code,
		message:   message,
		data:      data,
	}
}

func (err *RPCError) Error() string {
	id := "nil"
	if err.requestID != nil {
		id = strconv.Itoa(*err.requestID)
	}
	return fmt.Sprintf("RPC Error id: %s code: %d, err: %s, data: %v", id, err.code, err.message, err.data)
}

func (err *RPCError) toResponse() *ResponseErrorMessage {
	return &ResponseErrorMessage{
		JSONRPC: JSONRPCVersion,
		ID:      err.requestID,
		Error: ResponseError{
			Code:    err.code,
			Message: err.message,
			Data:    err.data,
		},
	}
}
