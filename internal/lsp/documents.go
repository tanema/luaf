package lsp

import (
	"errors"
	"strings"

	"github.com/tanema/luaf/internal/lerrors"
	"github.com/tanema/luaf/internal/parse"
)

type document struct {
	uri     string
	text    string
	fnProto *parse.FnProto
}

// openDocument keeps the previous successful parse on a syntax error, so
// definition/reference lookups keep working while mid-edit.
func (s *Server) openDocument(uri, text string) {
	fn, _, err := parse.Parse(uri, strings.NewReader(text), parse.ModeText)

	s.docsMu.Lock()
	doc := &document{uri: uri, text: text}
	if fn != nil {
		doc.fnProto = fn
	} else if prev, ok := s.documents[uri]; ok {
		doc.fnProto = prev.fnProto
	}
	s.documents[uri] = doc
	s.docsMu.Unlock()
	s.publishDiagnostics(uri, []Diagnostic{}, err)
}

func (s *Server) renameDocument(oldURI, newURI string) {
	s.docsMu.Lock()
	defer s.docsMu.Unlock()
	doc, ok := s.documents[oldURI]
	if !ok {
		return
	}
	delete(s.documents, oldURI)
	s.documents[newURI] = doc
}

func (s *Server) closeDocument(uri string) {
	s.docsMu.Lock()
	delete(s.documents, uri)
	s.docsMu.Unlock()
	s.publishDiagnostics(uri, []Diagnostic{}, nil)
}

func (s *Server) getDocument(uri string) *document {
	s.docsMu.Lock()
	defer s.docsMu.Unlock()
	return s.documents[uri]
}

func (s *Server) publishDiagnostics(uri string, diagnostics []Diagnostic, parseErr error) {
	if parseErr != nil {
		diagnostics = append(diagnostics, diagnosticFor(parseErr))
	}
	_ = s.notify("textDocument/publishDiagnostics", PublishDiagnosticsParams{URI: uri, Diagnostics: diagnostics})
}

func diagnosticFor(err error) Diagnostic {
	if lerr, ok := errors.AsType[*lerrors.Error](err); ok {
		pos := Position{Line: line0(lerr.Line), Character: col0(lerr.Column)}
		return Diagnostic{
			Range:    Range{Start: pos, End: Position{Line: pos.Line, Character: pos.Character + 1}},
			Severity: DiagnosticSeverityError,
			Source:   "luaf",
			Message:  lerr.Err.Error(),
		}
	}
	return Diagnostic{
		Severity: DiagnosticSeverityError,
		Source:   "luaf",
		Message:  err.Error(),
	}
}

// converts this parser's 1-based Line/Column into LSP's 0-based Position,
// clamping non-positive input to 0 instead of underflowing.
func line0(line int64) uint {
	if line <= 0 {
		return 0
	}
	return uint(line - 1)
}

func col0(column int64) uint {
	if column <= 0 {
		return 0
	}
	return uint(column - 1)
}

func (s *Server) definitionLocation(params GotoReq) *Location {
	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil || doc.fnProto == nil {
		return nil
	}
	ref := findReferenceAt(doc.fnProto, params.Position)
	if ref == nil || ref.Local == nil {
		return nil
	}
	loc := locationFor(params.TextDocument.URI, ref.Local.LineInfo, ref.Local.Name())
	return &loc
}

func (s *Server) referenceLocations(params ReferenceReq) []Location {
	locations := []Location{}
	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil || doc.fnProto == nil {
		return locations
	}
	target := resolveSymbolAt(doc.fnProto, params.Position)
	if target == nil {
		return locations
	}
	if params.Context.IncludeDeclaration {
		locations = append(locations, locationFor(params.TextDocument.URI, target.LineInfo, target.Name()))
	}
	for _, ref := range doc.fnProto.References {
		if ref.Local == target {
			locations = append(locations, locationFor(params.TextDocument.URI, ref.LineInfo, ref.Name))
		}
	}
	return locations
}

func locationFor(uri string, li parse.LineInfo, name string) Location {
	start := Position{Line: line0(li.Line), Character: col0(li.Column)}
	return Location{
		URI:   uri,
		Range: Range{Start: start, End: Position{Line: start.Line, Character: start.Character + uint(len(name))}},
	}
}

func findReferenceAt(fn *parse.FnProto, pos Position) *parse.Reference {
	for i, ref := range fn.References {
		if positionInSpan(ref.LineInfo, ref.Name, pos) {
			return &fn.References[i]
		}
	}
	return nil
}

func resolveSymbolAt(fn *parse.FnProto, pos Position) *parse.Local {
	if ref := findReferenceAt(fn, pos); ref != nil {
		return ref.Local
	}
	for _, lcl := range allLocals(fn) {
		if positionInSpan(lcl.LineInfo, lcl.Name(), pos) {
			return lcl
		}
	}
	return nil
}

func allLocals(fn *parse.FnProto) []*parse.Local {
	locals := append([]*parse.Local{}, fn.AllLocals...)
	for _, nested := range fn.FnTable {
		locals = append(locals, allLocals(nested)...)
	}
	return locals
}

// identifiers are single-line and ASCII, so a character-count span is exact.
func positionInSpan(li parse.LineInfo, name string, pos Position) bool {
	if name == "" || li.Line <= 0 {
		return false
	}
	start := col0(li.Column)
	return pos.Line == line0(li.Line) && pos.Character >= start && pos.Character < start+uint(len(name))
}
