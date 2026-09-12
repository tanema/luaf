package lsp

type (
	RequestMessage struct {
		JSONRPC string `json:"jsonrpc"`
		ID      *int   `json:"id"`
		Method  string `json:"method"`
		Payload []byte
	}
	ResponseError struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
		Data    any       `json:"data,omitempty"`
	}
	// Result has no omitempty: a success response must always carry a
	// result field, even null.
	ResponseMessage struct {
		JSONRPC string `json:"jsonrpc"`
		ID      *int   `json:"id"`
		Result  any    `json:"result"`
	}
	ResponseErrorMessage struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      *int          `json:"id"`
		Error   ResponseError `json:"error"`
	}
	WorkspaceFolder struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	}
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
				CodeDescriptionSupport *bool                `json:"CodeDescriptionSupport"`
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
	ServerInfo struct {
		Name    string  `json:"name"`
		Version *string `json:"version,omitempty"`
	}
	WorkspaceFoldersServerCapabilities struct {
		Supported           bool `json:"supported"`
		ChangeNotifications bool `json:"changeNotifications"`
	}
	FileOperationPatternOptions struct {
		IgnoreCase bool `json:"ignoreCase"`
	}
	FileOperationPattern struct {
		Glob    string                       `json:"glob"`
		Matches string                       `json:"matches,omitempty"`
		Options *FileOperationPatternOptions `json:"options,omitempty"`
	}
	FileOperationFilter struct {
		Scheme  string               `json:"scheme,omitempty"`
		Pattern FileOperationPattern `json:"pattern"`
	}
	FileOperationRegistrationOptions struct {
		Filters []FileOperationFilter `json:"filters,omitempty"`
	}
	ServerWorkspaceFileOperationsCapabilities struct {
		DidCreate  *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
		WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`
		DidRename  *FileOperationRegistrationOptions `json:"didRename,omitempty"`
		WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`
		DidDelete  *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
		WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
	}
	ServerWorkspaceCapabilities struct {
		WorkspaceFolders *WorkspaceFoldersServerCapabilities        `json:"workspaceFolders,omitempty"`
		FileOperations   *ServerWorkspaceFileOperationsCapabilities `json:"fileOperations,omitempty"`
	}
	ServerTextDocumentSync struct {
		OpenClose bool                 `json:"openClose"`
		Change    TextDocumentSyncKind `json:"change"`
	}
	CompletionOptions struct {
		TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
		AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`
		ResolveProvider     bool     `json:"resolveProvider"`
	}
	SignatureHelpOptions struct {
		TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
		RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
	}
	CodeActionOptions struct {
		CodeActionKind  []CodeActionKind `json:"codeActionKind,omitempty"`
		ResolveProvider bool             `json:"resolveProvider"`
	}
	CodeLensOptions struct {
		ResolveProvider bool `json:"resolveProvider"`
	}
	DocumentLinkOptions struct {
		ResolveProvider bool `json:"resolveProvider"`
	}
	DocumentOnTypeFormattingOptions struct {
		FirstTriggerCharacter string   `json:"firstTriggerCharacter,omitempty"`
		MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
	}
	ExecuteCommandOptions struct {
		Commands []string `json:"commands,omitempty"`
	}
	SemanticTokensLegend struct {
		TokenTypes     []string `json:"tokenTypes,omitempty"`
		TokenModifiers []string `json:"tokenModifiers,omitempty"`
	}
	SemanticTokensOptions struct {
		Legend SemanticTokensLegend `json:"legend"`
		Range  bool                 `json:"range"`
		Full   bool                 `json:"full"`
	}
	DiagnosticOptions struct {
		Identifier            string `json:"identifier,omitempty"`
		InterFileDependencies bool   `json:"interFileDependencies"`
		WorkspaceDiagnostics  bool   `json:"workspaceDiagnostics"`
	}
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
		DeclarationProvide               bool                             `json:"declarationProvide"`
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
	InitializeReq struct {
		ProcessID             *int               `json:"processId"`
		ClientInfo            *ServerInfo        `json:"clientInfo,omitempty"`
		Locale                *string            `json:"locale"`
		InitializationOptions map[string]any     `json:"initializationOptions"`
		Capabilities          ClientCapabilities `json:"capabilities"`
		Trace                 *string            `json:"trace"`
		WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders"`
	}
	InitializeResult struct {
		Capabilities ServerCapabilities `json:"capabilities"`
		ServerInfo   ServerInfo         `json:"serverInfo"`
	}
	SetTraceReq struct {
		Value string `json:"value"`
	}
	CancelReq struct {
		ID int `json:"id"`
	}
	LogTraceReq struct {
		Message string `json:"message"`
		Verbose string `json:"verbose"`
	}
	NoopParams       struct{}
	TextDocumentItem struct {
		URI        string `json:"uri"`
		LanguageID string `json:"languageId"`
		Version    int    `json:"version"`
		Text       string `json:"text"`
	}
	TextDocumentIdentifier struct {
		URI string `json:"uri"`
	}
	VersionedTextDocumentIdentifier struct {
		URI     string `json:"uri"`
		Version int    `json:"version"`
	}
	Position struct {
		Line      uint `json:"line"`
		Character uint `json:"character"`
	}
	Range struct {
		Start Position `json:"start"`
		End   Position `json:"end"`
	}
	TextDocumentContentChangeEvent struct {
		Range       Range  `json:"range"`
		RangeLength *uint  `json:"rangeLength"`
		Text        string `json:"text"`
	}
	FileRename struct {
		OldURI string `json:"oldUri"`
		NewURI string `json:"newUri"`
	}
	ReferenceContext struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	}
	DidOpenTextDocumentReq struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}
	WillSaveTextDocumentReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Reason       TextDocumentSaveReason `json:"reason"`
	}
	DidChangeTextDocumentReq struct {
		TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
		ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
	}
	DidCloseTextDocumentReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	DidSaveTextDocumentReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Text         string                 `json:"text"`
	}
	RenameFilesReq struct {
		Files []FileRename `json:"files"`
	}
	FileDelete struct {
		URI string `json:"uri"`
	}
	DeleteFilesReq struct {
		Files []FileDelete `json:"files"`
	}
	GotoReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}
	ReferenceReq struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		Context      ReferenceContext       `json:"context"`
	}
	NotificationMessage struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}
	Location struct {
		URI   string `json:"uri"`
		Range Range  `json:"range"`
	}
	Diagnostic struct {
		Range    Range              `json:"range"`
		Severity DiagnosticSeverity `json:"severity,omitempty"`
		Source   string             `json:"source,omitempty"`
		Message  string             `json:"message"`
	}
	PublishDiagnosticsParams struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
)
