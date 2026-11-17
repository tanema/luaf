package lsp

type (
	// RequestMessage is an incoming JSON-RPC request or notification. Payload keeps the raw message so params
	// can be decoded per method.
	RequestMessage struct {
		JSONRPC string `json:"jsonrpc"`
		ID      *int   `json:"id"`
		Method  string `json:"method"`
		Payload []byte `json:"-"`
	}
	// ResponseMessage is a successful JSON-RPC response to a request.
	ResponseMessage struct {
		JSONRPC string `json:"jsonrpc"`
		ID      *int   `json:"id"`
		Result  any    `json:"result"`
	}
	// NotificationMessage is a server-initiated JSON-RPC notification that expects no response.
	NotificationMessage struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}
	// ResponseErrorMessage is a JSON-RPC response for a request that failed.
	ResponseErrorMessage struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      *int          `json:"id"`
		Error   ResponseError `json:"error"`
	}
	// ResponseError describes why a request failed.
	ResponseError struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
		Data    any       `json:"data,omitempty"`
	}
	// WorkspaceFolder is a workspace root the client has open.
	WorkspaceFolder struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	}
	// ClientCapabilities describes the features the client supports, sent during initialize.
	ClientCapabilities struct {
		Workspace *struct {
			ApplyEdit     *bool `json:"applyEdit"`
			WorkspaceEdit *struct {
				DocumentChanges         *bool    `json:"documentChanges"`
				ResourceOperations      []string `json:"resourceOperations"`
				FailureHandling         *string  `json:"failureHandling"`
				NormalizesLineEndings   *bool    `json:"normalizesLineEndings"`
				ChangeAnnotationSupport *struct {
					GroupsOnLabel *bool `json:"groupsOnLabel"`
				} `json:"changeAnnotationSupport"`
			} `json:"workspaceEdit"`
			DidChangeConfiguration *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"didChangeConfiguration"`
			DidChangeWatchedFiles *struct {
				DynamicRegistration    *bool `json:"dynamicRegistration"`
				RelativePatternSupport *bool `json:"relativePatternSupport"`
			} `json:"didChangeWatchedFiles"`
			Symbol *struct {
				DynamicRegistration *bool                 `json:"dynamicRegistration"`
				SymbolKind          *ValueSet[SymbolKind] `json:"symbolKind"`
				TagSupport          *ValueSet[SymbolTag]  `json:"tagSupport"`
				ResolveSupport      *struct {
					Properties []string `json:"properties"`
				} `json:"resolveSupport"`
			} `json:"symbol"`
			ExecuteCommand *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"executeCommand"`
			WorkspaceFolders *bool `json:"workspaceFolders"`
			Configuration    *bool `json:"configuration"`
			SemanticTokens   *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"semanticTokens"`
			CodeLens *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"codeLens"`
			FileOperations *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				DidCreate           *bool `json:"didCreate"`
				WillCreate          *bool `json:"willCreate"`
				DidRename           *bool `json:"didRename"`
				WillRename          *bool `json:"willRename"`
				DidDelete           *bool `json:"didDelete"`
				WillDelete          *bool `json:"willDelete"`
			} `json:"fileOperations"`
			InlineValue *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"inlineValue"`
			InlayHint *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"inlayHint"`
			Diagnostics *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"diagnostics"`
		} `json:"workspace"`
		TextDocument *struct {
			Synchronization *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				WillSave            *bool `json:"willSave"`
				WillSaveWaitUntil   *bool `json:"willSaveWaitUntil"`
				DidSave             *bool `json:"didSave"`
			} `json:"synchronization"`
			Completion *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				CompletionItem      *struct {
					SnippetSupport          *bool                `json:"snippetSupport"`
					CommitCharactersSupport *bool                `json:"commitCharactersSupport"`
					DocumentationFormat     []string             `json:"documentationFormat"`
					DeprecatedSupport       *bool                `json:"deprecatedSupport"`
					PreselectSupport        *bool                `json:"preselectSupport"`
					TagSupport              *ValueSet[SymbolTag] `json:"tagSupport"`
					InsertReplaceSupport    *bool                `json:"insertReplaceSupport"`
					ResolveSupport          *struct {
						Properties []string `json:"properties"`
					} `json:"resolveSupport"`
					InsertTextModeSupport *ValueSet[InsertTextMode] `json:"insertTextModeSupport"`
					LabelDetailsSupport   *bool                     `json:"labelDetailsSupport"`
				} `json:"completionItem"`
				CompletionItemKind *ValueSet[CompletionItemKind] `json:"completionItemKind"`
				ContextSupport     *bool                         `json:"contextSupport"`
				InsertTextMode     *InsertTextMode               `json:"insertTextMode"`
				CompletionList     *struct {
					ItemDefaults []string `json:"itemDefaults"`
				} `json:"completionList"`
			} `json:"completion"`
			Hover *struct {
				DynamicRegistration *bool    `json:"dynamicRegistration"`
				ContentFormat       []string `json:"contentFormat"`
			} `json:"hover"`
			SignatureHelp *struct {
				DynamicRegistration  *bool `json:"dynamicRegistration"`
				SignatureInformation *struct {
					DocumentationFormat  []string `json:"documentationFormat"`
					ParameterInformation *struct {
						LabelOffsetSupport *bool `json:"labelOffsetSupport"`
					} `json:"parameterInformation"`
					ActiveParameterSupport *bool `json:"activeParameterSupport"`
				} `json:"signatureInformation"`
				ContextSupport *bool `json:"contextSupport"`
			} `json:"signatureHelp"`
			Declaration *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				LinkSupport         *bool `json:"linkSupport"`
			} `json:"declaration"`
			Definition *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				LinkSupport         *bool `json:"linkSupport"`
			} `json:"definition"`
			TypeDefinition *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				LinkSupport         *bool `json:"linkSupport"`
			} `json:"typeDefinition"`
			Implementation *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				LinkSupport         *bool `json:"linkSupport"`
			} `json:"implementation"`
			References *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"references"`
			DocumentHighlight *struct {
				RefreshSupport *bool `json:"refreshSupport"`
			} `json:"documentHighlight"`
			DocumentSymbol *struct {
				DynamicRegistration               *bool                 `json:"dynamicRegistration"`
				SymbolKind                        *ValueSet[SymbolKind] `json:"symbolKind"`
				HierarchicalDocumentSymbolSupport *bool                 `json:"hierarchicalDocumentSymbolSupport"`
				TagSupport                        *ValueSet[SymbolTag]  `json:"tagSupport"`
				LabelSupport                      *bool                 `json:"labelSupport"`
			} `json:"documentSymbol"`
			CodeAction *struct {
				DynamicRegistration      *bool `json:"dynamicRegistration"`
				CodeActionLiteralSupport *struct {
					CodeActionKind *ValueSet[CodeActionKind] `json:"codeActionKind"`
				} `json:"codeActionLiteralSupport"`
				IsPreferredSupport *bool `json:"isPreferredSupport"`
				DisabledSupport    *bool `json:"disabledSupport"`
				DataSupport        *bool `json:"dataSupport"`
				ResolveSupport     *struct {
					Properties []string `json:"properties"`
				} `json:"resolveSupport"`
				HonorsChangeAnnotations *bool `json:"honorsChangeAnnotations"`
			} `json:"codeAction"`
			CodeLens *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"codeLens"`
			DocumentLink *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				TooltipSupport      *bool `json:"tooltipSupport"`
			} `json:"documentLink"`
			ColorProvider *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"colorProvider"`
			Formatting *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"formatting"`
			RangeFormatting *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"rangeFormatting"`
			OnTypeFormatting *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"onTypeFormatting"`
			Rename *struct {
				DynamicRegistration           *bool `json:"dynamicRegistration"`
				PrepareSupport                *bool `json:"prepareSupport"`
				PrepareSupportDefaultBehavior *int  `json:"prepareSupportDefaultBehavior"`
				HonorsChangeAnnotations       *bool `json:"honorsChangeAnnotations"`
			} `json:"rename"`
			PublishDiagnostics *struct {
				RelatedInformation     *bool                `json:"relatedInformation"`
				TagSupport             *ValueSet[SymbolTag] `json:"tagSupport"`
				VersionSupport         *bool                `json:"versionSupport"`
				CodeDescriptionSupport *bool                `json:"codeDescriptionSupport"`
				DataSupport            *bool                `json:"dataSupport"`
			} `json:"publishDiagnostics"`
			FoldingRange *struct {
				DynamicRegistration *bool                       `json:"dynamicRegistration"`
				RangeLimit          *uint                       `json:"rangeLimit"`
				LineFoldingOnly     *bool                       `json:"lineFoldingOnly"`
				FoldingRangeKind    *ValueSet[FoldingRangeKind] `json:"foldingRangeKind"`
				FoldingRange        *struct {
					CollapsedText *bool `json:"collapsedText"`
				} `json:"foldingRange"`
			} `json:"foldingRange"`
			SelectionRange *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"selectionRange"`
			LinkedEditingRange *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"linkedEditingRange"`
			CallHierarchy *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"callHierarchy"`
			SemanticTokens *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				Requests            struct {
					Range any `json:"range"`
					Full  any `json:"full"`
				} `json:"requests"`
				TokenTypes              []string `json:"tokenTypes"`
				TokenModifiers          []string `json:"tokenModifiers"`
				Formats                 []string `json:"formats"`
				OverlappingTokenSupport *bool    `json:"overlappingTokenSupport"`
				MultilineTokenSupport   *bool    `json:"multilineTokenSupport"`
				ServerCancelSupport     *bool    `json:"serverCancelSupport"`
				AugmentsSyntaxTokens    *bool    `json:"augmentsSyntaxTokens"`
			} `json:"semanticTokens"`
			Moniker *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"moniker"`
			TypeHierarchy *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"typeHierarchy"`
			InlineValue *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
			} `json:"inlineValue"`
			InlayHint *struct {
				DynamicRegistration *bool `json:"dynamicRegistration"`
				ResolveSupport      *struct {
					Properties []string `json:"properties"`
				} `json:"resolveSupport"`
			} `json:"inlayHint"`
			Diagnostic *struct {
				DynamicRegistration    *bool                `json:"dynamicRegistration"`
				RelatedDocumentSupport *bool                `json:"relatedDocumentSupport"`
				RelatedInformation     *bool                `json:"relatedInformation"`
				TagSupport             *ValueSet[SymbolTag] `json:"tagSupport"`
				CodeDescriptionSupport *bool                `json:"codeDescriptionSupport"`
				DataSupport            *bool                `json:"dataSupport"`
			} `json:"diagnostic"`
		} `json:"textDocument"`
		NotebookDocument *struct {
			Synchronization struct {
				DynamicRegistration     *bool `json:"dynamicRegistration"`
				ExecutionSummarySupport *bool `json:"executionSummarySupport"`
			} `json:"synchronization"`
		} `json:"notebookDocument"`
		Window *struct {
			WorkDoneProgress *bool `json:"workDoneProgress"`
			ShowMessage      *struct {
				MessageActionItem *struct {
					AdditionalPropertiesSupport *bool `json:"additionalPropertiesSupport"`
				} `json:"messageActionItem"`
			} `json:"showMessage"`
			ShowDocument *struct {
				Support *bool `json:"support"`
			} `json:"showDocument"`
		} `json:"window"`
		General *struct {
			StaleRequestSupport *struct {
				Cancel                 *bool    `json:"cancel"`
				RetryOnContentModified []string `json:"retryOnContentModified"`
			} `json:"staleRequestSupport"`
			RegularExpressions *struct {
				Engine  string  `json:"engine"`
				Version *string `json:"version"`
			} `json:"regularExpressions"`
			Markdown *struct {
				Parser      string   `json:"parser"`
				Version     *string  `json:"version"`
				AllowedTags []string `json:"allowedTags"`
			} `json:"markdown"`
			PositionEncodings []PositionEncodingKind `json:"positionEncodings"`
		} `json:"general"`
		Experimental any `json:"experimental"`
	}
	// ServerInfo identifies a server or client by name and optional version.
	ServerInfo struct {
		Name    string  `json:"name"`
		Version *string `json:"version,omitempty"`
	}
	// WorkspaceFoldersServerCapabilities describes the server's support for multiple workspace folders.
	WorkspaceFoldersServerCapabilities struct {
		Supported           bool `json:"supported"`
		ChangeNotifications bool `json:"changeNotifications"`
	}
	// FileOperationPatternOptions configures how a file operation glob is matched.
	FileOperationPatternOptions struct {
		IgnoreCase bool `json:"ignoreCase"`
	}
	// FileOperationPattern is a glob that selects files or folders for file operation events.
	FileOperationPattern struct {
		Glob    string                       `json:"glob"`
		Matches string                       `json:"matches,omitempty"`
		Options *FileOperationPatternOptions `json:"options,omitempty"`
	}
	// FileOperationFilter selects files for file operation events by URI scheme and pattern.
	FileOperationFilter struct {
		Scheme  string               `json:"scheme,omitempty"`
		Pattern FileOperationPattern `json:"pattern"`
	}
	// FileOperationRegistrationOptions lists the filters a file operation event applies to.
	FileOperationRegistrationOptions struct {
		Filters []FileOperationFilter `json:"filters,omitempty"`
	}
	// ServerWorkspaceFileOperationsCapabilities lists the file create, rename and delete events the server
	// wants.
	ServerWorkspaceFileOperationsCapabilities struct {
		DidCreate  *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
		WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`
		DidRename  *FileOperationRegistrationOptions `json:"didRename,omitempty"`
		WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`
		DidDelete  *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
		WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
	}
	// ServerWorkspaceCapabilities describes the workspace features the server supports.
	ServerWorkspaceCapabilities struct {
		WorkspaceFolders *WorkspaceFoldersServerCapabilities        `json:"workspaceFolders,omitempty"`
		FileOperations   *ServerWorkspaceFileOperationsCapabilities `json:"fileOperations,omitempty"`
	}
	// ServerTextDocumentSync describes how the client should sync open documents with the server.
	ServerTextDocumentSync struct {
		OpenClose bool                 `json:"openClose"`
		Change    TextDocumentSyncKind `json:"change"`
	}
	// CompletionOptions configures the server's completion support.
	CompletionOptions struct {
		TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
		AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`
		ResolveProvider     bool     `json:"resolveProvider"`
	}
	// SignatureHelpOptions configures the server's signature help support.
	SignatureHelpOptions struct {
		TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
		RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
	}
	// CodeActionOptions configures the server's code action support.
	CodeActionOptions struct {
		CodeActionKind  []CodeActionKind `json:"codeActionKind,omitempty"`
		ResolveProvider bool             `json:"resolveProvider"`
	}
	// CodeLensOptions configures the server's code lens support.
	CodeLensOptions struct {
		ResolveProvider bool `json:"resolveProvider"`
	}
	// DocumentLinkOptions configures the server's document link support.
	DocumentLinkOptions struct {
		ResolveProvider bool `json:"resolveProvider"`
	}
	// DocumentOnTypeFormattingOptions lists the characters that trigger formatting while typing.
	DocumentOnTypeFormattingOptions struct {
		FirstTriggerCharacter string   `json:"firstTriggerCharacter,omitempty"`
		MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
	}
	// ExecuteCommandOptions lists the commands the server can execute.
	ExecuteCommandOptions struct {
		Commands []string `json:"commands,omitempty"`
	}
	// SemanticTokensLegend lists the token types and modifiers used in semantic token responses.
	SemanticTokensLegend struct {
		TokenTypes     []string `json:"tokenTypes,omitempty"`
		TokenModifiers []string `json:"tokenModifiers,omitempty"`
	}
	// SemanticTokensOptions configures the server's semantic token support.
	SemanticTokensOptions struct {
		Legend SemanticTokensLegend `json:"legend"`
		Range  bool                 `json:"range"`
		Full   bool                 `json:"full"`
	}
	// DiagnosticOptions configures the server's pull diagnostic support.
	DiagnosticOptions struct {
		Identifier            string `json:"identifier,omitempty"`
		InterFileDependencies bool   `json:"interFileDependencies"`
		WorkspaceDiagnostics  bool   `json:"workspaceDiagnostics"`
	}
	// ServerCapabilities describes the features the server supports, returned from initialize.
	ServerCapabilities struct {
		PositionEncoding                 *PositionEncodingKind            `json:"positionEncoding,omitempty"`
		TextDocumentSync                 *ServerTextDocumentSync          `json:"textDocumentSync,omitempty"`
		Workspace                        *ServerWorkspaceCapabilities     `json:"workspace,omitempty"`
		NotebookDocumentSync             map[string]string                `json:"notebookDocumentSync,omitempty"`
		CompletionProvider               *CompletionOptions               `json:"completionProvider,omitempty"`
		SignatureHelpProvider            *SignatureHelpOptions            `json:"signatureHelpProvider,omitempty"`
		CodeActionProvider               *CodeActionOptions               `json:"codeActionProvider,omitempty"`
		CodeLensProvider                 *CodeLensOptions                 `json:"codeLensProvider,omitempty"`
		DocumentLinkProvider             *DocumentLinkOptions             `json:"documentLinkProvider,omitempty"`
		DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
		ExecuteCommandProvider           *ExecuteCommandOptions           `json:"executeCommandProvider,omitempty"`
		SemanticTokensProvider           *SemanticTokensOptions           `json:"semanticTokensProvider,omitempty"`
		DiagnosticProvider               *DiagnosticOptions               `json:"diagnosticProvider,omitempty"`
		HoverProvider                    bool                             `json:"hoverProvider"`
		DeclarationProvider              bool                             `json:"declarationProvider"`
		DefinitionProvider               bool                             `json:"definitionProvider"`
		TypeDefinitionProvider           bool                             `json:"typeDefinitionProvider"`
		ImplementationProvider           bool                             `json:"implementationProvider"`
		ReferencesProvider               bool                             `json:"referencesProvider"`
		DocumentHighlightProvider        bool                             `json:"documentHighlightProvider"`
		DocumentSymbolProvider           bool                             `json:"documentSymbolProvider"`
		ColorProvider                    bool                             `json:"colorProvider"`
		DocumentFormattingProvider       bool                             `json:"documentFormattingProvider"`
		DocumentRangeFormattingProvider  bool                             `json:"documentRangeFormattingProvider"`
		RenameProvider                   bool                             `json:"renameProvider"`
		FoldingRangeProvider             bool                             `json:"foldingRangeProvider"`
		SelectionRangeProvider           bool                             `json:"selectionRangeProvider"`
		LinkedEditingRangeProvider       bool                             `json:"linkedEditingRangeProvider"`
		CallHierarchyProvider            bool                             `json:"callHierarchyProvider"`
		MonikerProvider                  bool                             `json:"monikerProvider"`
		TypeHierarchyProvider            bool                             `json:"typeHierarchyProvider"`
		InlineValueProvider              bool                             `json:"inlineValueProvider"`
		InlayHintProvider                bool                             `json:"inlayHintProvider"`
		WorkspaceSymbolProvider          bool                             `json:"workspaceSymbolProvider"`
		Experimental                     any                              `json:"experimental"`
	}
	// InitializeReq holds the params of the initialize request.
	InitializeReq struct {
		ProcessID             *int               `json:"processId"`
		ClientInfo            *ServerInfo        `json:"clientInfo,omitempty"`
		Locale                *string            `json:"locale"`
		InitializationOptions map[string]any     `json:"initializationOptions"`
		Capabilities          ClientCapabilities `json:"capabilities"`
		Trace                 *string            `json:"trace"`
		WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders"`
	}
	// InitializeResult is the response to the initialize request.
	InitializeResult struct {
		Capabilities ServerCapabilities `json:"capabilities"`
		ServerInfo   ServerInfo         `json:"serverInfo"`
	}
	// SetTraceReq holds the params of the $/setTrace notification.
	SetTraceReq struct {
		Value string `json:"value"`
	}
	// CancelReq holds the params of the $/cancelRequest notification.
	CancelReq struct {
		ID int `json:"id"`
	}
	// LogTraceReq holds the params of the $/logTrace notification.
	LogTraceReq struct {
		Message string `json:"message"`
		Verbose string `json:"verbose"`
	}
	// TextDocumentItem is a full text document sent when it is opened.
	TextDocumentItem struct {
		URI        string `json:"uri"`
		LanguageID string `json:"languageId"`
		Version    int    `json:"version"`
		Text       string `json:"text"`
	}
	// TextDocumentIdentifier identifies a text document by URI.
	TextDocumentIdentifier struct {
		URI string `json:"uri"`
	}
	// VersionedTextDocumentIdentifier identifies a specific version of a text document.
	VersionedTextDocumentIdentifier struct {
		URI     string `json:"uri"`
		Version int    `json:"version"`
	}
	// Position is a zero-based line and character offset in a text document.
	Position struct {
		Line      uint `json:"line"`
		Character uint `json:"character"`
	}
	// Range is the span between two positions in a text document.
	Range struct {
		Start Position `json:"start"`
		End   Position `json:"end"`
	}
	// TextDocumentContentChangeEvent describes a change to a text document's content.
	TextDocumentContentChangeEvent struct {
		Range       Range  `json:"range"`
		RangeLength *uint  `json:"rangeLength"`
		Text        string `json:"text"`
	}
	// FileRename is a single file rename in a rename files request.
	FileRename struct {
		OldURI string `json:"oldUri"`
		NewURI string `json:"newUri"`
	}
	// ReferenceContext configures whether a references request includes the declaration.
	ReferenceContext struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	}
	// DidOpenTextDocumentReq holds the params of the textDocument/didOpen notification.
	DidOpenTextDocumentReq struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}
	// WillSaveTextDocumentReq holds the params of the textDocument/willSave notification and willSaveWaitUntil
	// request.
	WillSaveTextDocumentReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Reason       TextDocumentSaveReason `json:"reason"`
	}
	// DidChangeTextDocumentReq holds the params of the textDocument/didChange notification.
	DidChangeTextDocumentReq struct {
		TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
		ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
	}
	// DidCloseTextDocumentReq holds the params of the textDocument/didClose notification.
	DidCloseTextDocumentReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	// DidSaveTextDocumentReq holds the params of the textDocument/didSave notification.
	DidSaveTextDocumentReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Text         string                 `json:"text"`
	}
	// RenameFilesReq holds the params of the workspace willRenameFiles and didRenameFiles messages.
	RenameFilesReq struct {
		Files []FileRename `json:"files"`
	}
	// FileDelete is a single file deletion in a delete files request.
	FileDelete struct {
		URI string `json:"uri"`
	}
	// DeleteFilesReq holds the params of the workspace willDeleteFiles and didDeleteFiles messages.
	DeleteFilesReq struct {
		Files []FileDelete `json:"files"`
	}
	// GotoReq holds the params of goto requests such as definition, declaration and implementation.
	GotoReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}
	// ReferenceReq holds the params of the textDocument/references request.
	ReferenceReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		Context      ReferenceContext       `json:"context"`
	}
	// Location is a range inside a document, identified by URI.
	Location struct {
		URI   string `json:"uri"`
		Range Range  `json:"range"`
	}
	// Diagnostic is a problem, such as an error or warning, reported for a range in a document.
	Diagnostic struct {
		Range    Range              `json:"range"`
		Severity DiagnosticSeverity `json:"severity,omitempty"`
		Source   string             `json:"source,omitempty"`
		Message  string             `json:"message"`
	}
	// PublishDiagnosticsParams holds the params of the textDocument/publishDiagnostics notification.
	PublishDiagnosticsParams struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
)
