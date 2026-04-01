package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	kai "github.com/tinhvqbk/kai"
	"github.com/tinhvqbk/kai/internal/brain"
	"github.com/tinhvqbk/kai/internal/config"
	"github.com/tinhvqbk/kai/internal/prompt"
	"github.com/tinhvqbk/kai/internal/provider"
	"github.com/tinhvqbk/kai/internal/retriever"
	"github.com/tinhvqbk/kai/internal/router"
	"github.com/tinhvqbk/kai/internal/session"
	"github.com/tinhvqbk/kai/internal/sheets"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Source labels for brain file provenance.
const (
	SourceMCP          = "mcp"
	SourceDirect       = "direct"
	SourceGoogleSheets = "google-sheets"
)

// DefaultConfidence is used when no confidence level is specified.
const DefaultConfidence = "medium"

type Server struct {
	mcp      *mcpserver.MCPServer
	router   *router.Router
	brain    *brain.FileBrain
	sessions *session.Manager
	provider provider.Provider
	builder  *prompt.Builder
	cfg      *config.Config
	sheets   *sheets.Client
}

func NewServer(
	r *router.Router,
	b *brain.FileBrain,
	sm *session.Manager,
	p provider.Provider,
	cfg *config.Config,
) *Server {
	s := &Server{
		router:   r,
		brain:    b,
		sessions: sm,
		provider: p,
		builder:  prompt.NewBuilder(),
		cfg:      cfg,
	}

	if cfg.Google.ServiceAccountPath != "" {
		sc, err := sheets.NewClient(context.Background(), cfg.Google.ServiceAccountPath)
		if err != nil {
			log.Printf("WARNING: Google Sheets unavailable: %v", err)
		} else {
			s.sheets = sc
		}
	}

	s.mcp = mcpserver.NewMCPServer(
		"kai",
		kai.Version,
		mcpserver.WithToolCapabilities(false),
	)

	s.registerTools()
	return s
}

func (s *Server) Serve() error {
	return mcpserver.ServeStdio(s.mcp)
}

// ServeHTTP starts the MCP server over Streamable HTTP with bearer token auth.
func (s *Server) ServeHTTP(addr, token string) error {
	httpServer := mcpserver.NewStreamableHTTPServer(s.mcp)
	expected := []byte("Bearer " + token)

	mux := http.NewServeMux()
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, expected) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		httpServer.ServeHTTP(w, r)
	}))

	log.Printf("MCP server listening on %s/mcp", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) registerTools() {
	s.mcp.AddTool(
		mcplib.NewTool("brain_list",
			mcplib.WithDescription("List all brain knowledge files with tags and confidence levels"),
		),
		s.handleBrainList,
	)

	s.mcp.AddTool(
		mcplib.NewTool("brain_read",
			mcplib.WithDescription("Read a specific brain file's full content and metadata"),
			mcplib.WithString("path",
				mcplib.Required(),
				mcplib.Description("Relative path within brain directory (e.g. 'opinions/testing.md')"),
			),
		),
		s.handleBrainRead,
	)

	s.mcp.AddTool(
		mcplib.NewTool("brain_write",
			mcplib.WithDescription("Write or update a brain file directly. No LLM needed. Creates the file if it doesn't exist."),
			mcplib.WithString("path",
				mcplib.Required(),
				mcplib.Description("Relative path within brain directory (e.g. 'opinions/testing.md')"),
			),
			mcplib.WithString("content",
				mcplib.Required(),
				mcplib.Description("The content to write"),
			),
			mcplib.WithString("tags",
				mcplib.Required(),
				mcplib.Description("Comma-separated tags (e.g. 'go,testing,tdd')"),
			),
			mcplib.WithString("confidence",
				mcplib.Description("Confidence level: high, medium, or low (default: medium)"),
			),
		),
		s.handleBrainWrite,
	)

	s.mcp.AddTool(
		mcplib.NewTool("brain_search",
			mcplib.WithDescription("Search brain knowledge by keywords. Returns scored results. No LLM call."),
			mcplib.WithString("query",
				mcplib.Required(),
				mcplib.Description("Search keywords (e.g. 'testing go patterns')"),
			),
		),
		s.handleBrainSearch,
	)

	s.mcp.AddTool(
		mcplib.NewTool("ask",
			mcplib.WithDescription("Ask kai a question. Two modes: (1) With LLM: full pipeline — retrieval, prompt, LLM call, learning extraction. Returns answer. (2) Without LLM (e.g. via Claude Code): retrieves brain context and builds system prompt. Returns the context so the MCP client can answer as the user's digital twin."),
			mcplib.WithString("question",
				mcplib.Required(),
				mcplib.Description("The question to ask"),
			),
		),
		s.handleAsk,
	)

	s.mcp.AddTool(
		mcplib.NewTool("teach",
			mcplib.WithDescription("Teach kai something new. Two modes: (1) With LLM configured: pass 'input' and kai extracts learnings automatically. (2) Without LLM (e.g. via Claude Code): pass 'category', 'topic', 'content', 'tags' directly — you do the extraction, kai saves it."),
			mcplib.WithString("input",
				mcplib.Description("Free-form teaching input (requires LLM provider). e.g. 'I prefer TDD for business logic'"),
			),
			mcplib.WithString("category",
				mcplib.Description("Brain category: identity, opinions, style, decisions, or knowledge (direct mode, no LLM needed)"),
			),
			mcplib.WithString("topic",
				mcplib.Description("Topic slug for filename, e.g. 'testing-philosophy' (direct mode)"),
			),
			mcplib.WithString("content",
				mcplib.Description("The knowledge content in markdown (direct mode)"),
			),
			mcplib.WithString("tags",
				mcplib.Description("Comma-separated tags, e.g. 'testing,tdd,go' (direct mode)"),
			),
			mcplib.WithString("confidence",
				mcplib.Description("Confidence level: high, medium, or low (default: medium)"),
			),
			mcplib.WithString("action",
				mcplib.Description("Action: create, append, or replace (default: create)"),
			),
		),
		s.handleTeach,
	)

	s.mcp.AddTool(
		mcplib.NewTool("brain_reorganize",
			mcplib.WithDescription("Analyze brain files for duplicates, miscategorizations, and consolidation opportunities. Returns a reorganization plan. Pass apply=true to execute it."),
			mcplib.WithString("mode",
				mcplib.Description("Mode: all, dedup, recategorize, or consolidate (default: all)"),
			),
			mcplib.WithNumber("similarity_threshold",
				mcplib.Description("Similarity threshold for duplicate detection 0.0-1.0 (default: 0.7)"),
			),
			mcplib.WithBoolean("apply",
				mcplib.Description("Execute the plan immediately (default: false, dry-run)"),
			),
		),
		s.handleBrainReorganize,
	)

	s.mcp.AddTool(
		mcplib.NewTool("sheet_read",
			mcplib.WithDescription("Read a Google Sheet and save it to brain. Returns markdown table. The sheet must be shared with the service account."),
			mcplib.WithString("spreadsheet_id",
				mcplib.Required(),
				mcplib.Description("The spreadsheet ID from the Google Sheets URL"),
			),
			mcplib.WithString("range",
				mcplib.Description("Optional A1 range (e.g. 'Sheet1!A1:D10'). Defaults to entire first sheet."),
			),
			mcplib.WithString("brain_path",
				mcplib.Description("Optional brain path to save the data (e.g. 'data/my-sheet.md'). If omitted, data is returned but not saved."),
			),
			mcplib.WithString("tags",
				mcplib.Description("Comma-separated tags for brain storage (e.g. 'bugs,tracking'). Used only when brain_path is set."),
			),
		),
		s.handleSheetRead,
	)
}

func (s *Server) handleBrainList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	files, err := s.brain.ListAll()
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("list brain files: %v", err)), nil
	}

	type brainEntry struct {
		Path       string   `json:"path"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
	}

	var entries []brainEntry
	for _, f := range files {
		bf, err := s.brain.Load(f)
		if err != nil {
			continue
		}
		entries = append(entries, brainEntry{
			Path:       f,
			Tags:       bf.Tags,
			Confidence: bf.Confidence,
		})
	}

	if entries == nil {
		entries = []brainEntry{}
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleBrainSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: query"), nil
	}

	type searchResult struct {
		Path    string  `json:"path"`
		Score   float64 `json:"score"`
		Preview string  `json:"preview"`
	}

	// Full-text TF-IDF search with relevance scoring
	type candidate struct {
		path  string
		score float64
	}
	var candidates []candidate

	fullResults := s.brain.QueryFullIndex(query)
	if len(fullResults) > 0 {
		for _, fr := range fullResults {
			candidates = append(candidates, candidate{path: fr.Path, score: fr.Score})
		}
	} else {
		// Fall back to tag-based search
		keywords := strings.Fields(strings.ToLower(query))
		tagResults := s.brain.QueryIndex(keywords)
		for _, tr := range tagResults {
			candidates = append(candidates, candidate{path: tr.Path, score: float64(tr.Score)})
		}
	}

	var entries []searchResult
	for _, c := range candidates {
		bf, err := s.brain.Load(c.path)
		if err != nil {
			continue
		}
		finalScore := brain.RelevanceScore(bf, c.score)
		preview := ""
		content := strings.TrimSpace(bf.Content)
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) > 0 {
			preview = lines[0]
		}
		entries = append(entries, searchResult{
			Path:    c.path,
			Score:   finalScore,
			Preview: preview,
		})
	}

	// Sort by score descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})

	if entries == nil {
		entries = []searchResult{}
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleAsk(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	question, err := req.RequireString("question")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: question"), nil
	}

	// With LLM: full pipeline
	if s.router != nil {
		sess := s.sessions.New(s.cfg.Provider)

		chatResult, err := s.router.Chat(ctx, sess, question)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("chat error: %v", err)), nil
		}

		s.sessions.Save(sess)

		type askResponse struct {
			Response       string   `json:"response"`
			Confidence     string   `json:"confidence"`
			BrainFilesUsed []string `json:"brain_files_used"`
		}

		data, err := json.Marshal(askResponse{
			Response:       chatResult.Response,
			Confidence:     chatResult.Confidence,
			BrainFilesUsed: chatResult.BrainFilesUsed,
		})
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
		}

		return mcplib.NewToolResultText(string(data)), nil
	}

	// Without LLM: retrieve brain context and return it for the MCP client to use
	ret := retriever.New(s.brain, s.cfg.Brain.MaxContextFiles)
	brainFiles, err := ret.Retrieve(ctx, question, s.cfg.Brain.MaxContextFiles)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("retrieve brain context: %v", err)), nil
	}

	systemPrompt := s.builder.BuildSystemPrompt(brainFiles)

	var brainPaths []string
	for _, bf := range brainFiles {
		brainPaths = append(brainPaths, bf.Path)
	}

	type contextResponse struct {
		SystemPrompt   string   `json:"system_prompt"`
		BrainFilesUsed []string `json:"brain_files_used"`
		Instruction    string   `json:"instruction"`
	}

	data, err := json.Marshal(contextResponse{
		SystemPrompt:   systemPrompt,
		BrainFilesUsed: brainPaths,
		Instruction:    "Use the system_prompt as your personality and context to answer the user's question as their digital twin.",
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleBrainRead(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: path"), nil
	}

	bf, err := s.brain.Load(path)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("load brain file: %v", err)), nil
	}

	type brainFileResponse struct {
		Path       string   `json:"path"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
		Content    string   `json:"content"`
	}

	data, err := json.Marshal(brainFileResponse{
		Path:       bf.Path,
		Tags:       bf.Tags,
		Confidence: bf.Confidence,
		Content:    strings.TrimSpace(bf.Content),
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleBrainWrite(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: path"), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: content"), nil
	}

	tagsStr, err := req.RequireString("tags")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: tags"), nil
	}

	confidence, _ := req.RequireString("confidence")
	if confidence == "" {
		confidence = DefaultConfidence
	}

	var tags []string
	for _, t := range strings.Split(tagsStr, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	bf := &brain.BrainFile{
		Path:       path,
		Tags:       tags,
		Confidence: confidence,
		Source:     SourceMCP,
		Updated:    brain.DateOnly{Time: time.Now()},
		Content:    "\n" + content + "\n",
	}

	if err := s.brain.Save(bf); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("save brain file: %v", err)), nil
	}
	if err := s.brain.RebuildIndex(); err != nil {
		log.Printf("WARNING: rebuild brain index: %v", err)
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Saved %s", path)), nil
}

func (s *Server) handleSheetRead(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	if s.sheets == nil {
		return mcplib.NewToolResultError("Google Sheets not configured — set google.service_account_path in config.yaml"), nil
	}

	spreadsheetID, err := req.RequireString("spreadsheet_id")
	if err != nil {
		return mcplib.NewToolResultError("missing required parameter: spreadsheet_id"), nil
	}

	readRange, _ := req.RequireString("range")
	brainPath, _ := req.RequireString("brain_path")
	tagsStr, _ := req.RequireString("tags")

	result, err := s.sheets.Read(ctx, spreadsheetID, readRange)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("read sheet: %v", err)), nil
	}

	// Save to brain if path provided.
	if brainPath != "" {
		tags := []string{SourceGoogleSheets}
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				tags = append(tags, strings.TrimSpace(t))
			}
		}

		content := fmt.Sprintf("# %s\n\nSource: Google Sheet `%s` range `%s`\n\n%s",
			result.Title, spreadsheetID, result.Range, result.Markdown)

		bf := &brain.BrainFile{
			Path:       brainPath,
			Tags:       tags,
			Confidence: "high",
			Source:     SourceGoogleSheets,
			Updated:    brain.DateOnly{Time: time.Now()},
			Content:    "\n" + content + "\n",
		}

		if err := s.brain.Save(bf); err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("save to brain: %v", err)), nil
		}
		if err := s.brain.RebuildIndex(); err != nil {
		log.Printf("WARNING: rebuild brain index: %v", err)
	}
	}

	// Return response.
	type sheetResponse struct {
		Title     string `json:"title"`
		Range     string `json:"range"`
		RowCount  int    `json:"row_count"`
		BrainPath string `json:"brain_path,omitempty"`
		Markdown  string `json:"markdown"`
	}

	data, err := json.Marshal(sheetResponse{
		Title:     result.Title,
		Range:     result.Range,
		RowCount:  len(result.Rows),
		BrainPath: brainPath,
		Markdown:  result.Markdown,
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleBrainReorganize(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	mode, _ := req.RequireString("mode")
	if mode == "" {
		mode = brain.ModeAll
	}

	apply := false
	if v, err := req.RequireBool("apply"); err == nil {
		apply = v
	}

	simThreshold := 0.7
	if v, err := req.RequireFloat("similarity_threshold"); err == nil && v > 0 {
		simThreshold = v
	}

	opts := brain.ReorgOptions{
		Mode:                mode,
		SimilarityThreshold: simThreshold,
		SmallFileThreshold:  50,
		DryRun:              !apply,
	}

	// Reorganize holds a single lock for the entire find+apply sequence,
	// preventing TOCTOU races between analysis and execution.
	plan, err := s.brain.Reorganize(opts)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("reorganize: %v", err)), nil
	}

	type planResponse struct {
		Actions []brain.ReorgAction `json:"actions"`
		Applied bool                `json:"applied"`
		Count   int                 `json:"count"`
	}

	actions := plan.Actions
	if actions == nil {
		actions = []brain.ReorgAction{}
	}

	data, err := json.Marshal(planResponse{
		Actions: actions,
		Applied: apply,
		Count:   len(actions),
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleTeach(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// Direct mode: category + topic + content provided (no LLM needed)
	category, _ := req.RequireString("category")
	topic, _ := req.RequireString("topic")
	content, _ := req.RequireString("content")

	if category != "" && topic != "" && content != "" {
		return s.handleTeachDirect(category, topic, content, req)
	}

	// LLM mode: free-form input, extract learnings via provider
	input, _ := req.RequireString("input")
	if input == "" {
		return mcplib.NewToolResultError("provide either 'input' (requires LLM) or 'category' + 'topic' + 'content' (direct mode, no LLM)"), nil
	}

	if s.provider == nil {
		return mcplib.NewToolResultError("free-form 'input' requires an LLM provider. Either configure a provider in config.yaml, or use direct mode with 'category', 'topic', and 'content' parameters"), nil
	}

	messages := []provider.Message{
		{Role: "user", Content: "I want to teach you about myself: " + input},
	}

	teachPrompt := s.builder.BuildLearningPrompt([]string{}, messages)

	resp, err := s.provider.Chat(ctx, "", []provider.Message{
		{Role: "user", Content: teachPrompt},
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("LLM error: %v", err)), nil
	}

	learnings, err := brain.ParseLearnings(resp.Content)
	if err != nil || len(learnings) == 0 {
		return mcplib.NewToolResultText("I couldn't extract any specific learnings from that. Try being more concrete."), nil
	}

	for i := range learnings {
		learnings[i].Source = SourceDirect
	}

	if err := s.brain.Learn(learnings); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("save learnings: %v", err)), nil
	}
	if err := s.brain.RebuildIndex(); err != nil {
		log.Printf("WARNING: rebuild brain index: %v", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Learned %d new things:\n", len(learnings)))
	for _, l := range learnings {
		sb.WriteString(fmt.Sprintf("  - %s/%s.md\n", l.Category, l.Topic))
	}

	return mcplib.NewToolResultText(sb.String()), nil
}

func (s *Server) handleTeachDirect(category, topic, content string, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	tagsStr, _ := req.RequireString("tags")
	confidence, _ := req.RequireString("confidence")
	action, _ := req.RequireString("action")

	if confidence == "" {
		confidence = DefaultConfidence
	}
	if action == "" {
		action = "create"
	}

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	learnings := []brain.Learning{{
		Category:   category,
		Topic:      topic,
		Tags:       tags,
		Content:    content,
		Confidence: confidence,
		Action:     action,
		Source:     SourceDirect,
	}}

	if err := s.brain.Learn(learnings); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("save learning: %v", err)), nil
	}
	if err := s.brain.RebuildIndex(); err != nil {
		log.Printf("WARNING: rebuild brain index: %v", err)
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Saved to brain/%s/%s.md", category, topic)), nil
}
