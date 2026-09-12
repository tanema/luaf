package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer() (*Server, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return &Server{
		documents: make(map[string]*document),
		inflight:  make(map[int]context.CancelFunc),
		ctx:       context.Background(),
		out:       buf,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, buf
}

// run with -race; previously panicked with "concurrent map writes" on s.inflight.
func TestDispatchHandlerConcurrentAccess(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer()

	var wg sync.WaitGroup
	for i := range 200 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			srv.dispatchHandler(&id, "initialized", []byte(`{"params":{}}`))
		}(i)
	}
	wg.Wait()
}

func lastMessage(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	data := buf.Bytes()
	idx := bytes.LastIndex(data, []byte("\r\n\r\n"))
	require.GreaterOrEqual(t, idx, 0, "no framed message found in buffer")
	var msg map[string]any
	require.NoError(t, json.Unmarshal(data[idx+4:], &msg))
	return msg
}

func decodeInto(t *testing.T, v any, out any) {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, out))
}

func TestDocDidOpenPublishesDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("valid source publishes no diagnostics", func(t *testing.T) {
		t.Parallel()
		srv, buf := newTestServer()
		require.NoError(t, srv.docDidOpen(context.Background(), nil, DidOpenTextDocumentReq{
			TextDocument: TextDocumentItem{URI: "file:///t.luaf", Text: "local x = 1"},
		}))

		msg := lastMessage(t, buf)
		assert.Equal(t, "textDocument/publishDiagnostics", msg["method"])
		var params PublishDiagnosticsParams
		decodeInto(t, msg["params"], &params)
		assert.Empty(t, params.Diagnostics)
	})

	t.Run("invalid source publishes a diagnostic at the error position", func(t *testing.T) {
		t.Parallel()
		srv, buf := newTestServer()
		require.NoError(t, srv.docDidOpen(context.Background(), nil, DidOpenTextDocumentReq{
			TextDocument: TextDocumentItem{URI: "file:///t.luaf", Text: "local x = "},
		}))

		msg := lastMessage(t, buf)
		var params PublishDiagnosticsParams
		decodeInto(t, msg["params"], &params)
		require.Len(t, params.Diagnostics, 1)
		assert.Equal(t, DiagnosticSeverityError, params.Diagnostics[0].Severity)
	})

	t.Run("a syntax error keeps the previous successful parse available", func(t *testing.T) {
		t.Parallel()
		srv, buf := newTestServer()
		require.NoError(t, srv.docDidOpen(context.Background(), nil, DidOpenTextDocumentReq{
			TextDocument: TextDocumentItem{URI: "file:///t.luaf", Text: "local x = 1\nprint(x)"},
		}))
		buf.Reset()

		require.NoError(t, srv.docDidChange(context.Background(), nil, DidChangeTextDocumentReq{
			TextDocument:   VersionedTextDocumentIdentifier{URI: "file:///t.luaf"},
			ContentChanges: []TextDocumentContentChangeEvent{{Text: "local x = 1\nprint(x"}},
		}))

		doc := srv.getDocument("file:///t.luaf")
		require.NotNil(t, doc)
		require.NotNil(t, doc.fnProto, "last good parse should be retained across a syntax error")
	})
}

func TestDocDefinition(t *testing.T) {
	t.Parallel()

	srv, buf := newTestServer()
	require.NoError(t, srv.docDidOpen(context.Background(), nil, DidOpenTextDocumentReq{
		TextDocument: TextDocumentItem{URI: "file:///t.luaf", Text: "local x = 1\nprint(x)"},
	}))
	buf.Reset()

	t.Run("jumps from a usage to its declaration", func(t *testing.T) {
		id := 1
		require.NoError(t, srv.docDefinition(context.Background(), &id, GotoReq{
			TextDocument: TextDocumentIdentifier{URI: "file:///t.luaf"},
			Position:     Position{Line: 1, Character: 6}, // the 'x' in print(x)
		}))
		msg := lastMessage(t, buf)
		var loc Location
		decodeInto(t, msg["result"], &loc)
		assert.Equal(t, "file:///t.luaf", loc.URI)
		assert.Equal(t, Position{Line: 0, Character: 6}, loc.Range.Start) // the 'x' in local x = 1
	})

	t.Run("returns a null result for a global", func(t *testing.T) {
		id := 2
		require.NoError(t, srv.docDefinition(context.Background(), &id, GotoReq{
			TextDocument: TextDocumentIdentifier{URI: "file:///t.luaf"},
			Position:     Position{Line: 1, Character: 0}, // 'print'
		}))
		msg := lastMessage(t, buf)
		result, hasResult := msg["result"]
		assert.True(t, hasResult, "a request must always get a response, even a null one")
		assert.Nil(t, result)
	})

	t.Run("returns a null result for an unknown document", func(t *testing.T) {
		id := 3
		require.NoError(t, srv.docDefinition(context.Background(), &id, GotoReq{
			TextDocument: TextDocumentIdentifier{URI: "file:///missing.luaf"},
			Position:     Position{Line: 0, Character: 0},
		}))
		msg := lastMessage(t, buf)
		result, hasResult := msg["result"]
		assert.True(t, hasResult)
		assert.Nil(t, result)
	})
}

func TestDocReferences(t *testing.T) {
	t.Parallel()

	srv, buf := newTestServer()
	require.NoError(t, srv.docDidOpen(context.Background(), nil, DidOpenTextDocumentReq{
		TextDocument: TextDocumentItem{URI: "file:///t.luaf", Text: "local x = 1\nprint(x)\nprint(x)"},
	}))
	buf.Reset()

	t.Run("from the declaration, with declaration included", func(t *testing.T) {
		id := 1
		require.NoError(t, srv.docReferences(context.Background(), &id, ReferenceReq{
			TextDocument: TextDocumentIdentifier{URI: "file:///t.luaf"},
			Position:     Position{Line: 0, Character: 6}, // the 'x' in local x = 1
			Context:      ReferenceContext{IncludeDeclaration: true},
		}))
		msg := lastMessage(t, buf)
		var locs []Location
		decodeInto(t, msg["result"], &locs)
		require.Len(t, locs, 3) // declaration + 2 usages
	})

	t.Run("from a usage, without the declaration", func(t *testing.T) {
		id := 2
		require.NoError(t, srv.docReferences(context.Background(), &id, ReferenceReq{
			TextDocument: TextDocumentIdentifier{URI: "file:///t.luaf"},
			Position:     Position{Line: 1, Character: 6}, // the 'x' in the first print(x)
			Context:      ReferenceContext{IncludeDeclaration: false},
		}))
		msg := lastMessage(t, buf)
		var locs []Location
		decodeInto(t, msg["result"], &locs)
		require.Len(t, locs, 2)
	})
}

func TestRequestStubsAlwaysRespond(t *testing.T) {
	t.Parallel()
	srv, buf := newTestServer()
	id := 1

	require.NoError(t, srv.docTypeDefinition(context.Background(), &id, GotoReq{}))
	msg := lastMessage(t, buf)
	result, hasResult := msg["result"]
	assert.True(t, hasResult)
	assert.Nil(t, result)

	require.NoError(t, srv.docWillSaveWaitUntil(context.Background(), &id, WillSaveTextDocumentReq{}))
	msg = lastMessage(t, buf)
	_, hasResult = msg["result"]
	assert.True(t, hasResult)

	require.NoError(t, srv.workspaceWillRenameFiles(context.Background(), &id, RenameFilesReq{}))
	msg = lastMessage(t, buf)
	_, hasResult = msg["result"]
	assert.True(t, hasResult)

	require.NoError(t, srv.workspaceWillDeleteFiles(context.Background(), &id, DeleteFilesReq{}))
	msg = lastMessage(t, buf)
	_, hasResult = msg["result"]
	assert.True(t, hasResult)
}
