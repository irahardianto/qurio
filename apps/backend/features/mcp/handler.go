package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"sync"
	"time"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/middleware"
	"github.com/google/uuid"
)

type Retriever interface {
	Search(ctx context.Context, query string, opts *retrieval.SearchOptions) ([]retrieval.SearchResult, error)
	GetChunksByURL(ctx context.Context, url string) ([]retrieval.SearchResult, error)
}

type SourceManager interface {
	List(ctx context.Context) ([]source.Source, error)
	GetPages(ctx context.Context, id string) ([]source.SourcePage, error)
}

type Handler struct {
	retriever    Retriever
	sourceMgr    SourceManager
	sessions     map[string]chan string // sessionId -> message channel (serialized JSON-RPC response)
	sessionsLock sync.RWMutex
}

func NewHandler(r Retriever, s SourceManager) *Handler {
	return &Handler{
		retriever: r,
		sourceMgr: s,
		sessions:  make(map[string]chan string),
	}
}

// JSON-RPC Request types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type CallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type SearchArgs struct {
	Query    string                 `json:"query"`
	Alpha    *float32               `json:"alpha,omitempty"`
	Limit    *int                   `json:"limit,omitempty"`
	SourceID *string                `json:"source_id,omitempty"`
	Filters  map[string]interface{} `json:"filters,omitempty"`
}

type FetchPageArgs struct {
	URL string `json:"url"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// JSON-RPC Response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// processRequest processes the JSON-RPC request and returns a response.
// Returns nil if no response should be sent (e.g. for notifications).
func (h *Handler) processRequest(ctx context.Context, req JSONRPCRequest) *JSONRPCResponse {
	if req.Method == "initialize" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "qurio-mcp",
					"version": "1.0.0",
				},
			},
		}
	}

	if req.Method == "notifications/initialized" {
		// Notifications must not generate a response
		return nil
	}

	if req.Method == "tools/list" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ListToolsResult{
				Tools: []Tool{
					{
						Name:        "qurio_search",
						Description: `Search & Exploration tool. Performs a hybrid search (Keyword + Vector). Use this for specific questions, finding code snippets, or exploring topics across known sources.

ARGUMENT GUIDE:

[Alpha: Hybrid Search Balance]
- 0.0 (Keyword): Use for Error Codes ("0x8004"), IDs ("550e8400"), or unique strings.
- 0.3 (Mostly Keyword): Use for specific function names ("handle_web_task") where exact match matters but context helps.
- 0.5 (Hybrid - Default): Safe bet for general queries like "database configuration".
- 1.0 (Vector): Use for conceptual "How do I..." questions (e.g. "stop server" matches "shutdown").

[Limit: Result Count]
- Default: 10
- Recommended: 5-15 (Prevent context bloat)
- Max: 50

[Filters: Metadata Filtering]
- type: Filter by content type (e.g., "code", "prose", "api", "config").
- language: Filter by language (e.g., "go", "python", "json").

USAGE EXAMPLES:
- Specific: search(query="webhook signature", alpha=0.3)
- Conceptual: search(query="how to handle errors", alpha=1.0)
- Filtered: search(query="User struct", filters={"type": "code", "language": "go"})`,
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]string{
									"type":        "string",
									"description": "The search query",
								},
								"alpha": map[string]interface{}{
									"type":        "number",
									"description": "Hybrid search balance (0.0=Keyword, 1.0=Vector). See tool description for guide.",
									"minimum":     0.0,
									"maximum":     1.0,
								},
								"limit": map[string]interface{}{
									"type":        "integer",
									"description": "Max results to return (default 10).",
									"minimum":     1,
									"maximum":     50,
								},
								"source_id": map[string]string{
									"type":        "string",
									"description": "Filter results by source ID",
								},
								"filters": map[string]interface{}{
									"type":        "object",
									"description": "Metadata filters (e.g. type='code', language='go')",
								},
							},
							"required": []string{"query"},
						},
					},
					{
						Name:        "qurio_list_sources",
						Description: `Discovery tool. Lists all available documentation sets (sources) currently indexed. Use this at the start of a session to understand what documentation is available.

USAGE EXAMPLE:
qurio_list_sources()`,
						InputSchema: map[string]interface{}{
							"type":       "object",
							"properties": map[string]interface{}{},
						},
					},
					{
						Name:        "qurio_list_pages",
						Description: `Navigation tool. Lists all individual pages/documents within a specific source. Use this to find the exact URL of a document when a search query is too broad or to browse the table of contents.

USAGE EXAMPLE:
qurio_list_pages(source_id="src_stripe_api")`,
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"source_id": map[string]string{
									"type":        "string",
									"description": "The ID of the source",
								},
							},
							"required": []string{"source_id"},
						},
					},
					{
						Name:        "qurio_read_page",
						Description: `Deep Reading / Full Context tool. Retrieves the *entire* content of a specific page or document by its URL. Use this when a search result snippet is truncated or insufficient, or when you need to read a full guide/tutorial. Crucial: Always prefer this over guessing content if the search result is incomplete.

USAGE EXAMPLE:
read_page(url="https://docs.stripe.com/webhooks/signatures")`,
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"url": map[string]string{
									"type":        "string",
									"description": "The URL to fetch content for",
								},
							},
							"required": []string{"url"},
						},
					},
				},
			},
		}
	}

	if req.Method == "tools/call" {
		var params CallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			slog.Warn("invalid params structure", "error", err)
			resp := makeErrorResponse(req.ID, ErrInvalidParams, "Invalid params")
			return &resp
		}

		if params.Name == "qurio_search" || params.Name == "search" { // Backward compatibility for now? Or strict? Plan says "Rename". Strict is better to verify change.
			// Actually, let's stick to strict renaming as per plan.
			if params.Name == "search" {
				// Optional: Support alias or reject. Plan says "Rename".
				// I will treat "search" as not found or alias?
				// To be safe and strict: "Rename" implies old name is gone.
			}
		}

		if params.Name == "qurio_search" {
			var args SearchArgs
			if err := json.Unmarshal(params.Arguments, &args); err != nil {
				slog.Warn("invalid search arguments", "error", err)
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "Invalid search arguments")
				return &resp
			}
			
			if args.Query == "" {
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "Query is required")
				return &resp
			}

			if args.Alpha != nil && (*args.Alpha < 0.0 || *args.Alpha > 1.0) {
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "Alpha must be between 0.0 and 1.0")
				return &resp
			}

			if args.SourceID != nil && *args.SourceID != "" {
				if args.Filters == nil {
					args.Filters = make(map[string]interface{})
				}
				args.Filters["sourceId"] = *args.SourceID
			}

			opts := &retrieval.SearchOptions{
				Alpha:   args.Alpha,
				Limit:   args.Limit,
				Filters: args.Filters,
			}
			results, err := h.retriever.Search(ctx, args.Query, opts)
			if err != nil {
				slog.Error("search failed", "error", err)
				resp := makeErrorResponse(req.ID, ErrInternal, "Search failed: "+err.Error())
				return &resp
			}
			
			var textResult string
			if len(results) == 0 {
				textResult = "No results found."
			} else {
				for i, res := range results {
					textResult += fmt.Sprintf("Result %d (Score: %.2f):\n", i+1, res.Score)
					if res.Title != "" {
						textResult += fmt.Sprintf("Title: %s\n", res.Title)
					}
					// Extract Type, Language, and SourceID from explicit fields
					if res.Type != "" {
						textResult += fmt.Sprintf("Type: %s\n", res.Type)
					}
					if res.Language != "" {
						textResult += fmt.Sprintf("Language: %s\n", res.Language)
					}
					if res.SourceID != "" {
						textResult += fmt.Sprintf("SourceID: %s\n", res.SourceID)
					}
					
					textResult += fmt.Sprintf("Content:\n%s\n", res.Content)
					
					// Optional: Show other metadata
					// if len(res.Metadata) > 0 {
					// 	meta, _ := json.Marshal(res.Metadata)
					// 	textResult += fmt.Sprintf("Metadata: %s\n", string(meta))
					// }
					textResult += "\n---\n"
				}
				
				textResult += "\nUse qurio_read_page(url=\"...\") to read the full content of any result.\n"
			}

			slog.Info("tool execution completed", "tool", "qurio_search", "result_count", len(results))

			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolResult{
					Content: []ToolContent{
						{Type: "text", Text: textResult},
					},
				},
			}
		}

		if params.Name == "qurio_list_sources" {
			sources, err := h.sourceMgr.List(ctx)
			if err != nil {
				slog.Error("list_sources failed", "error", err)
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}},
						IsError: true,
					},
				}
			}

			if len(sources) == 0 {
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{
							{Type: "text", Text: "No sources found."},
						},
					},
				}
			}

			type SimpleSource struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			}
			
			simpleSources := make([]SimpleSource, len(sources))
			for i, s := range sources {
				name := s.Name
				if name == "" {
					name = s.URL
				}
				simpleSources[i] = SimpleSource{
					ID:   s.ID,
					Name: name,
					Type: s.Type,
				}
			}

			jsonBytes, err := json.MarshalIndent(simpleSources, "", "  ")
			if err != nil {
				slog.Error("failed to marshal sources", "error", err)
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{{Type: "text", Text: "Error marshalling results"}},
						IsError: true,
					},
				}
			}

			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolResult{
					Content: []ToolContent{
						{Type: "text", Text: string(jsonBytes)},
					},
				},
			}
		}

		if params.Name == "qurio_list_pages" {
			type ListPagesArgs struct {
				SourceID string `json:"source_id"`
			}
			var args ListPagesArgs
			if err := json.Unmarshal(params.Arguments, &args); err != nil {
				slog.Error("invalid arguments for list_pages", "error", err)
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "Invalid arguments")
				return &resp
			}
			
			if args.SourceID == "" {
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "source_id is required")
				return &resp
			}

			pages, err := h.sourceMgr.GetPages(ctx, args.SourceID)
			if err != nil {
				slog.Error("list_pages failed", "error", err)
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}},
						IsError: true,
					},
				}
			}

			if len(pages) == 0 {
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{
							{Type: "text", Text: "No pages found for source."},
						},
					},
				}
			}

			type SimplePage struct {
				ID  string `json:"id"`
				URL string `json:"url"`
			}
			
			simplePages := make([]SimplePage, len(pages))
			for i, p := range pages {
				simplePages[i] = SimplePage{
					ID:  p.ID,
					URL: p.URL,
				}
			}

			jsonBytes, err := json.MarshalIndent(simplePages, "", "  ")
			if err != nil {
				slog.Error("failed to marshal pages", "error", err)
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{{Type: "text", Text: "Error marshalling results"}},
						IsError: true,
					},
				}
			}

			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolResult{
					Content: []ToolContent{
						{Type: "text", Text: string(jsonBytes)},
					},
				},
			}
		}

		if params.Name == "qurio_read_page" {
			var args FetchPageArgs
			if err := json.Unmarshal(params.Arguments, &args); err != nil {
				slog.Warn("invalid read_page arguments", "error", err)
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "Invalid arguments")
				return &resp
			}
			
			if args.URL == "" {
				resp := makeErrorResponse(req.ID, ErrInvalidParams, "URL is required")
				return &resp
			}

			results, err := h.retriever.GetChunksByURL(ctx, args.URL)
			if err != nil {
				slog.Error("read_page failed", "error", err)
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ToolResult{
						Content: []ToolContent{{Type: "text", Text: "Error: " + err.Error()}},
						IsError: true,
					},
				}
			}

			var textResult string
			if len(results) == 0 {
				textResult = "No content found for URL."
			} else {
				title := ""
				if len(results) > 0 {
					title = results[0].Title
				}
				textResult = fmt.Sprintf("Page: %s\nURL: %s\n\n", title, args.URL)
				for _, res := range results {
					if res.Type == "code" {
						textResult += fmt.Sprintf("[Code Block: %s]\n%s\n\n", res.Language, res.Content)
					} else {
						textResult += fmt.Sprintf("%s\n\n", res.Content)
					}
				}
			}

			slog.Info("tool execution completed", "tool", "qurio_read_page", "chunk_count", len(results))

			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolResult{
					Content: []ToolContent{
						{Type: "text", Text: textResult},
					},
				},
			}
		}
		
		slog.Warn("method not found", "method", params.Name)
		resp := makeErrorResponse(req.ID, ErrMethodNotFound, "Method not found: "+params.Name)
		return &resp
	}
	
	slog.Warn("unknown jsonrpc method", "method", req.Method)
	resp := makeErrorResponse(req.ID, ErrMethodNotFound, "Method not found")
	return &resp
}

func makeErrorResponse(id interface{}, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Error: map[string]interface{}{
			"code":    code,
			"message": message,
		},
		ID: id,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("mcp request received", "method", r.Method, "path", r.URL.Path)
	
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, nil, ErrParse, "Parse error")
		return
	}

	resp := h.processRequest(r.Context(), req)
	if resp != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		// Notification, just return OK
		w.WriteHeader(http.StatusOK)
	}
}

// HandleSSE establishes the SSE connection and manages the session
func (h *Handler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// 1. Set SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 2. Create Session
	sessionID := uuid.New().String()
	msgChan := make(chan string, 100) // Increased buffer to prevent drops

	h.sessionsLock.Lock()
	h.sessions[sessionID] = msgChan
	h.sessionsLock.Unlock()

	// Cleanup on disconnect
	defer func() {
		h.sessionsLock.Lock()
		delete(h.sessions, sessionID)
		h.sessionsLock.Unlock()
		close(msgChan)
		slog.Info("sse session ended", "session_id", sessionID)
	}()

	slog.Info("sse session started", "session_id", sessionID)

	// 3. Send 'endpoint' event
	// Construct absolute URL for client compatibility
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s/mcp/messages?sessionId=%s", scheme, r.Host, sessionID)
	safeEndpoint := html.EscapeString(endpoint)

	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", safeEndpoint)
	w.(http.Flusher).Flush()

	// Send 'id' event (Optional but good practice if client expects it)
	safeSessionID := html.EscapeString(sessionID)
	fmt.Fprintf(w, "event: id\ndata: %s\n\n", safeSessionID)
	w.(http.Flusher).Flush()

	// 4. Loop: Send messages from channel to SSE stream
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-ticker.C:
			// Send keep-alive comment to prevent timeouts
			fmt.Fprintf(w, ": keepalive\n\n")
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// HandleMessage accepts POST messages associated with a session
func (h *Handler) HandleMessage(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.GetCorrelationID(r.Context())
	
	slog.Info("mcp message received", 
		"method", r.Method, 
		"path", r.URL.Path,
		"correlation_id", correlationID,
	)

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		slog.Warn("missing sessionId in message request", "correlation_id", correlationID)
		h.writeHttpError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing sessionId", correlationID)
		return
	}

	h.sessionsLock.RLock()
	msgChan, exists := h.sessions[sessionID]
	h.sessionsLock.RUnlock()

	if !exists {
		slog.Warn("session not found", "session_id", sessionID, "correlation_id", correlationID)
		h.writeHttpError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", correlationID)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid json in message request", "error", err, "correlation_id", correlationID)
		h.writeHttpError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", correlationID)
		return
	}

	// MCP Spec: Return 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
	
	// Create detached context to preserve values (correlationID) but ignore cancellation
	bgCtx := context.WithoutCancel(r.Context())

	// Process asynchronously
	go func() {
		resp := h.processRequest(bgCtx, req)
		if resp == nil {
			// Notification, no response needed
			return
		}
		
		// Serialize response
		respBytes, err := json.Marshal(resp)
		if err != nil {
			slog.Error("failed to marshal response", "error", err, "correlation_id", correlationID)
			return
		}

		// Send to SSE channel safely
		h.sessionsLock.RLock()
		defer h.sessionsLock.RUnlock()
		
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("failed to send to sse channel (closed)", "session_id", sessionID, "error", r, "correlation_id", correlationID)
			}
		}()

		select {
		case msgChan <- string(respBytes):
		default:
			slog.Warn("session channel full, dropping message", "session_id", sessionID, "correlation_id", correlationID)
		}
	}()
}

func (h *Handler) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	// JSON-RPC errors are usually 200 OK at HTTP level, containing the error object
	// But some implementations use 400/500. We'll use 200 to be safe with clients 
	// that parse the body regardless of status, or 400/500 if strict HTTP semantics are needed.
	// Standard JSON-RPC over HTTP typically uses 200 OK.
	w.WriteHeader(http.StatusOK) 

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: map[string]interface{}{
			"code":    code,
			"message": message,
		},
		ID: id,
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) writeHttpError(w http.ResponseWriter, status int, code string, message string, correlationID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]interface{}{
		"status": "error",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
		"correlationId": correlationID,
	}
	json.NewEncoder(w).Encode(resp)
}