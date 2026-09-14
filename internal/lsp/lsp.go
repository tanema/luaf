// Package lsp contains a lua lsp functionality that reads from stdin and writes
// to stderr.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/tanema/luaf/internal/conf"
)

type Server struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	reader                *bufio.Reader
	out                   io.Writer
	logLevel              *slog.LevelVar
	logger                *slog.Logger
	initialized           bool
	pid                   int
	clientName            string
	clientVersion         string
	locale                string
	initializationOptions map[string]any
	capabilities          ClientCapabilities
	workspaceFolders      []WorkspaceFolder
	inflightMu            sync.Mutex
	inflight              map[int]context.CancelFunc
	outMu                 sync.Mutex
	docsMu                sync.Mutex
	documents             map[string]*document
}

func NewServer() *Server {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:       ctx,
		cancel:    cancel,
		reader:    bufio.NewReader(os.Stdin),
		out:       os.Stdout,
		logLevel:  lvl,
		inflight:  make(map[int]context.CancelFunc),
		documents: make(map[string]*document),
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: lvl,
		})),
	}
}

func (s *Server) Listen() error {
	for {
		req, err := s.read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.logger.Info("client closed the connection, shutting down")
				return nil
			}
			s.logger.Error("Error Reading Message", "error", err)
			continue
		}

		if req.Method == "exit" {
			s.cancel()
			os.Exit(0)
		}

		if !s.initialized && req.Method != "initialize" {
			if req.ID != nil {
				s.writeErr(rpcErr(req.ID, ServerNotInitialized, "Server not initialized", nil))
			}
			continue
		}

		go s.dispatchHandler(req.ID, req.Method, req.Payload)
	}
}

func (s *Server) read() (*RequestMessage, error) {
	contentLen := 0
	contentType := DefaultContentType
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			s.writeErr(rpcErr(nil, InvalidRequest, err.Error(), nil))
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.Split(line, ":")
		switch strings.TrimSpace(strings.ToLower(parts[0])) {
		case "content-length":
			n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				s.writeErr(rpcErr(nil, InvalidRequest, err.Error(), nil))
				return nil, err
			}
			contentLen = n
		case "content-type":
			contentType = strings.TrimSpace(parts[1])
		}
	}
	if contentType != DefaultContentType {
		s.writeErr(rpcErr(nil, InvalidRequest, "unsupported LSP content-type", nil))
		return nil, fmt.Errorf("unsupported LSP content-type: %s", contentType)
	}

	message := make([]byte, contentLen)
	_, err := io.ReadFull(s.reader, message)
	if err != nil {
		s.writeErr(rpcErr(nil, InvalidRequest, err.Error(), nil))
		return nil, err
	}

	req := &RequestMessage{Payload: message}
	if err = json.Unmarshal(message, req); err != nil {
		s.writeErr(rpcErr(nil, ParseError, err.Error(), nil))
		return nil, err
	}

	return req, nil
}

func (s *Server) writeResp(id *int, result any) error {
	return s.write(ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) writeErr(err *RPCError) error {
	return s.write(err.toResponse())
}

func (s *Server) notify(method string, params any) error {
	return s.write(NotificationMessage{JSONRPC: "2.0", Method: method, Params: params})
}

func (s *Server) write(resp any) error {
	message, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	contentLen := fmt.Sprintf("Content-Length: %d", len(message))
	contentType := fmt.Sprintf("Content-Type: %s", DefaultContentType)
	output := strings.Join([]string{contentLen, contentType, "", string(message)}, "\r\n")
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if _, err := s.out.Write([]byte(output)); err != nil {
		return err
	}
	return nil
}

func (s *Server) dispatchHandler(id *int, method string, payload []byte) {
	var reqErr error
	defer func() { s.errorHandler(id, method, reqErr) }()
	ctx := s.ctx
	if id != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		s.inflightMu.Lock()
		s.inflight[*id] = cancel
		s.inflightMu.Unlock()
		defer func() {
			s.inflightMu.Lock()
			delete(s.inflight, *id)
			s.inflightMu.Unlock()
		}()
	}

	switch method {
	case "initialize":
		reqErr = callHandler(ctx, id, payload, s.initialize)
	case "initialized":
		s.logger.Info("Received initialized", "req_id", id)
	case "shutdown":
		s.cancel()
	case "$/logTrace":
		reqErr = callHandler(ctx, id, payload, s.logTrace)
	case "$/setTrace":
		reqErr = callHandler(ctx, id, payload, s.setTrace)
	case "$/cancelRequest":
		reqErr = callHandler(ctx, id, payload, s.cancelRequest)
	case "$/progress":
	case "textDocument/didOpen":
		reqErr = callHandler(ctx, id, payload, s.docDidOpen)
	case "textDocument/didChange":
		reqErr = callHandler(ctx, id, payload, s.docDidChange)
	case "textDocument/willSave":
		reqErr = callHandler(ctx, id, payload, s.docWillSave)
	case "textDocument/willSaveWaitUntil":
		reqErr = callHandler(ctx, id, payload, s.docWillSaveWaitUntil)
	case "textDocument/didSave":
		reqErr = callHandler(ctx, id, payload, s.docDidSave)
	case "textDocument/didClose":
		reqErr = callHandler(ctx, id, payload, s.docDidClose)
	case "textDocument/declaration":
		reqErr = callHandler(ctx, id, payload, s.docDeclaration)
	case "textDocument/definition":
		reqErr = callHandler(ctx, id, payload, s.docDefinition)
	case "textDocument/typeDefinition":
		reqErr = callHandler(ctx, id, payload, s.docTypeDefinition)
	case "textDocument/implementation":
		reqErr = callHandler(ctx, id, payload, s.docImplementation)
	case "textDocument/references":
		reqErr = callHandler(ctx, id, payload, s.docReferences)
	case "workspace/willRenameFiles":
		reqErr = callHandler(ctx, id, payload, s.workspaceWillRenameFiles)
	case "workspace/didRenameFiles":
		reqErr = callHandler(ctx, id, payload, s.workspaceDidRenameFiles)
	case "workspace/willDeleteFiles":
		reqErr = callHandler(ctx, id, payload, s.workspaceWillDeleteFiles)
	case "workspace/didDeleteFiles":
		reqErr = callHandler(ctx, id, payload, s.workspaceDidDeleteFiles)
	default:
		s.logger.Warn("Received Unsupported RPC Request", "id", id, "method", method)
	}
}

func (s *Server) initialize(ctx context.Context, id *int, params InitializeReq) error {
	s.pid = FromPtr(params.ProcessID)
	s.locale = FromPtr(params.Locale)
	s.clientName = FromPtr(params.ClientInfo).Name
	s.clientVersion = FromPtr(FromPtr(params.ClientInfo).Version)
	s.initializationOptions = params.InitializationOptions
	s.capabilities = params.Capabilities
	if params.Trace != nil {
		s.setLogLevel(*params.Trace)
	}
	s.workspaceFolders = params.WorkspaceFolders
	s.initialized = true
	return s.writeResp(id, InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &ServerTextDocumentSync{
				OpenClose: true,
				Change:    TextDocumentSyncKindFull,
			},
			Workspace: &ServerWorkspaceCapabilities{
				WorkspaceFolders: &WorkspaceFoldersServerCapabilities{
					Supported:           true,
					ChangeNotifications: true,
				},
			},
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{".", ":"},
				ResolveProvider:   true,
			},
			SignatureHelpProvider: &SignatureHelpOptions{
				TriggerCharacters:   []string{"("},
				RetriggerCharacters: []string{","},
			},
		},
		ServerInfo: ServerInfo{
			Name:    "luaf-lsp",
			Version: ToPtr(conf.LUAVERSION),
		},
	})
}

func (s *Server) logTrace(ctx context.Context, id *int, params LogTraceReq) error {
	s.logger.Info("logTrace", "message", params.Message)
	s.logger.Debug("logTrace", "message", params.Verbose)
	return nil
}

func (s *Server) setTrace(ctx context.Context, id *int, params SetTraceReq) error {
	s.setLogLevel(params.Value)
	return nil
}

func (s *Server) cancelRequest(ctx context.Context, id *int, params CancelReq) error {
	s.inflightMu.Lock()
	cancel, ok := s.inflight[params.ID]
	s.inflightMu.Unlock()
	if ok {
		cancel()
	}
	return nil
}

func (s *Server) setLogLevel(trace string) {
	switch trace {
	case "off":
		s.logLevel.Set(slog.LevelError)
	case "messages":
		s.logLevel.Set(slog.LevelInfo)
	case "verbose":
		s.logLevel.Set(slog.LevelDebug)
	}
}

func (s *Server) docDidOpen(ctx context.Context, id *int, params DidOpenTextDocumentReq) error {
	s.openDocument(params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

func (s *Server) docDidChange(ctx context.Context, id *int, params DidChangeTextDocumentReq) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}
	text := params.ContentChanges[len(params.ContentChanges)-1].Text // full sync: last change is the whole document
	s.openDocument(params.TextDocument.URI, text)
	return nil
}

func (s *Server) docWillSave(ctx context.Context, id *int, params WillSaveTextDocumentReq) error {
	return nil
}

func (s *Server) docWillSaveWaitUntil(ctx context.Context, id *int, params WillSaveTextDocumentReq) error {
	return s.writeResp(id, nil)
}

func (s *Server) docDidSave(ctx context.Context, id *int, params DidSaveTextDocumentReq) error {
	return nil
}

func (s *Server) docDidClose(ctx context.Context, id *int, params DidCloseTextDocumentReq) error {
	s.closeDocument(params.TextDocument.URI)
	return nil
}

func (s *Server) docDeclaration(ctx context.Context, id *int, params GotoReq) error {
	return s.writeResp(id, s.definitionLocation(params))
}

func (s *Server) docDefinition(ctx context.Context, id *int, params GotoReq) error {
	return s.writeResp(id, s.definitionLocation(params))
}

func (s *Server) docTypeDefinition(ctx context.Context, id *int, params GotoReq) error {
	return s.writeResp(id, nil)
}

func (s *Server) docImplementation(ctx context.Context, id *int, params GotoReq) error {
	return s.writeResp(id, nil)
}

func (s *Server) docReferences(ctx context.Context, id *int, params ReferenceReq) error {
	return s.writeResp(id, s.referenceLocations(params))
}

func (s *Server) workspaceWillRenameFiles(ctx context.Context, id *int, params RenameFilesReq) error {
	// TODO: refactor require statements for this filename.
	return s.writeResp(id, nil)
}

func (s *Server) workspaceDidRenameFiles(ctx context.Context, id *int, params RenameFilesReq) error {
	for _, file := range params.Files {
		s.renameDocument(file.OldURI, file.NewURI)
	}
	return nil
}

func (s *Server) workspaceWillDeleteFiles(ctx context.Context, id *int, params DeleteFilesReq) error {
	// TODO: refactor? Can we remove requires?
	return s.writeResp(id, nil)
}

func (s *Server) workspaceDidDeleteFiles(ctx context.Context, id *int, params DeleteFilesReq) error {
	for _, file := range params.Files {
		s.closeDocument(file.URI)
	}
	return nil
}

func (s *Server) errorHandler(id *int, method string, reqErr error) {
	if reqErr == nil {
		return
	}

	s.logger.Error(
		"Error Handling Method",
		"id", FromPtr(id),
		"method", method,
		"error", reqErr,
	)
	if err, ok := errors.AsType[*RPCError](reqErr); ok {
		s.writeErr(err)
	} else {
		s.writeErr(rpcErr(id, RequestFailed, reqErr.Error(), nil))
	}
}
