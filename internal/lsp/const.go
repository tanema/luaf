package lsp

type (
	ErrorCode              int
	SymbolKind             int
	SymbolTag              int
	PositionEncodingKind   string
	InsertTextMode         int
	CompletionItemKind     int
	CodeActionKind         string
	FoldingRangeKind       string
	TextDocumentSyncKind   int
	TextDocumentSaveReason int
	DiagnosticSeverity     int
	ValueSet[T any]        struct {
		ValueSet []T `json:"valueSet"`
	}
)

const (
	DefaultContentType = "application/vscode-jsonrpc; charset=utf-8"

	// Defined by JSON-RPC
	ParseError     ErrorCode = -32700
	InvalidRequest ErrorCode = -32600
	MethodNotFound ErrorCode = -32601
	InvalidParams  ErrorCode = -32602
	InternalError  ErrorCode = -32603
	// This is the start range of JSON-RPC reserved error codes. It doesn't denote a
	// real error code. No LSP error codes should be defined between the start and
	// end range. For backwards compatibility the `ServerNotInitialized` and the
	// `UnknownErrorCode` are left in the range.
	JsonrpcReservedErrorRangeStart ErrorCode = -32099
	// ServerNotInitialized Error code indicating that a server received a notification or request before
	// the server received the `initialize` request.
	ServerNotInitialized ErrorCode = -32002
	UnknownErrorCode     ErrorCode = -32001
	// This is the end range of JSON-RPC reserved error codes. It doesn't denote a real error code.
	JsonrpcReservedErrorRangeEnd = -32000
	// This is the start range of LSP reserved error codes. It doesn't denote a real error code.
	LspReservedErrorRangeStart ErrorCode = -32899
	// RequestFailed A request failed but it was syntactically correct, e.g the method name was
	// known and the parameters were valid. The error message should contain human
	// readable information about why the request failed.
	RequestFailed ErrorCode = -32803
	// ServerCancelled The server cancelled the request. This error code should only be used for
	// requests that explicitly support being server cancellable.
	ServerCancelled ErrorCode = -32802
	// ContentModified The server detected that the content of a document got modified outside normal conditions. A server should
	// NOT send this error code if it detects a content change in its unprocessed messages. The result even computed
	// on an older state might still be useful for the client. If a client decides that a result is not of any use anymore
	// the client should cancel the request.
	ContentModified ErrorCode = -32801
	// RequestCancelled The client has canceled a request and a server has detected the cancel.
	RequestCancelled ErrorCode = -32800
	// LspReservedErrorRangeEnd is the end range of LSP reserved error codes. It doesn't denote a real error code.
	LspReservedErrorRangeEnd ErrorCode = -32800

	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26

	SymbolTagDeprecated SymbolTag = 1

	UTF8  PositionEncodingKind = "utf-8"
	UTF16 PositionEncodingKind = "utf-16"
	UTF32 PositionEncodingKind = "utf-32"

	// The insertion or replace strings is taken as it is. If the value is multi
	// line the lines below the cursor will be inserted using the indentation
	// defined in the string value. The client will not apply any kind of adjustments
	// to the string.
	InsertTextModeAsIs InsertTextMode = 1
	// The editor adjusts leading whitespace of new lines so that they match the
	// indentation up to the cursor of the line for which the item is accepted.
	// Consider a line like this: <2tabs><cursor><3tabs>foo. Accepting a multi line
	// completion item is indented using 2 tabs and all following lines inserted
	// will be indented using 2 tabs as well.
	InsertTextModeAdjustIndentation InsertTextMode = 2

	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25

	CodeActionKindEmpty                 CodeActionKind = ""
	CodeActionKindQuickFix              CodeActionKind = "quickfix"
	CodeActionKindRefactor              CodeActionKind = "refactor"
	CodeActionKindRefactorExtract       CodeActionKind = "refactor.extract"
	CodeActionKindRefactorInline        CodeActionKind = "refactor.inline"
	CodeActionKindRefactorRewrite       CodeActionKind = "refactor.rewrite"
	CodeActionKindSource                CodeActionKind = "source"
	CodeActionKindSourceOrganizeImports CodeActionKind = "source.organizeImports"
	CodeActionKindSourceFixAll          CodeActionKind = "source.fixAll"

	FoldingRangeKindComment FoldingRangeKind = "comment"
	FoldingRangeKindImports FoldingRangeKind = "imports"
	FoldingRangeKindRegion  FoldingRangeKind = "region"

	TextDocumentSyncKindNone        TextDocumentSyncKind = 0
	TextDocumentSyncKindFull        TextDocumentSyncKind = 1
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2

	TextDocumentSaveReasonManual     TextDocumentSaveReason = 1
	TextDocumentSaveReasonAfterDelay TextDocumentSaveReason = 2
	TextDocumentSaveReasonFocusOut   TextDocumentSaveReason = 3

	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)
