package dashboard

import (
	"crypto/rand"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	kai "github.com/norenis/kai"
	"github.com/norenis/kai/internal/brain"
	"github.com/norenis/kai/internal/config"
	"github.com/norenis/kai/internal/router"
	bsync "github.com/norenis/kai/internal/sync"
	"github.com/xuri/excelize/v2"
)

//go:embed templates/*
var templateFS embed.FS

// Server serves the brain dashboard.
type Server struct {
	brain        *brain.FileBrain
	brainPath    string
	cfg          *config.Config
	sessions     sync.Map // token → expiry (unix seconds)
	sessionsMu   sync.Mutex
	syncer       *bsync.Syncer
	router       *router.Router
}

// SetRouter sets the router for dashboard chat.
func (s *Server) SetRouter(r *router.Router) {
	s.router = r
}

// New creates a dashboard Server.
func New(b *brain.FileBrain, brainPath string, cfg *config.Config) *Server {
	s := &Server{brain: b, brainPath: brainPath, cfg: cfg}
	s.loadSessions()
	return s
}

func (s *Server) sessionsFile() string {
	return filepath.Join(s.brainPath, ".kai_sessions.json")
}

func (s *Server) loadSessions() {
	data, err := os.ReadFile(s.sessionsFile())
	if err != nil {
		return
	}
	var m map[string]int64
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	now := time.Now().Unix()
	for token, expiry := range m {
		if expiry > now {
			s.sessions.Store(token, expiry)
		}
	}
}

func (s *Server) saveSessions() {
	m := map[string]int64{}
	now := time.Now().Unix()
	s.sessions.Range(func(k, v any) bool {
		expiry, _ := v.(int64)
		if expiry > now {
			m[k.(string)] = expiry
		}
		return true
	})
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.sessionsFile(), data, 0600)
}

// SetSyncer sets the sync backend for dashboard sync controls.
func (s *Server) SetSyncer(syncer *bsync.Syncer) {
	s.syncer = syncer
}

// serviceStatus represents the status of a service connection.
type serviceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`    // "connected", "disconnected", "not_configured"
	Value   string `json:"value"`     // Primary display value (provider name, repo name, etc.)
	Tooltip string `json:"tooltip"`   // Extended info shown on hover
}

// ToolDef describes an available MCP tool for the dashboard.
type ToolDef struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Params      []ToolParam `json:"params"`
}

// ToolParam describes a tool parameter.
type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "number", "boolean"
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// ListenAndServe starts the dashboard as a standalone HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	return http.ListenAndServe(addr, mux)
}

// RegisterRoutes registers dashboard routes on the given mux at /.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// PIN auth endpoints (no auth required).
	mux.HandleFunc("/api/auth", s.handleAuth)

	// All other routes require PIN if configured.
	auth := s.withAuth
	mux.HandleFunc("/", auth(s.handleIndex))
	mux.HandleFunc("/api/overview", auth(s.handleOverview))
	mux.HandleFunc("/api/tasks", auth(s.handleTasks))
	mux.HandleFunc("/api/activity", auth(s.handleActivity))
	mux.HandleFunc("/api/search", auth(s.handleSearch))
	mux.HandleFunc("/api/info", auth(s.handleInfo))
	mux.HandleFunc("/api/tags", auth(s.handleTags))
	mux.HandleFunc("/api/files", auth(s.handleFiles))
	mux.HandleFunc("/api/file", auth(s.handleFileContent))
	mux.HandleFunc("/api/reorg", auth(s.handleReorg))
	mux.HandleFunc("/api/status", auth(s.handleStatus))
	mux.HandleFunc("/api/tools", auth(s.handleTools))
	mux.HandleFunc("/api/tools/execute", auth(s.handleToolExecute))
	mux.HandleFunc("/api/chat", auth(s.handleChat))
	mux.HandleFunc("/api/chat/stream", auth(s.handleChatStream))
	mux.HandleFunc("/api/import", auth(s.handleImport))
	mux.HandleFunc("/api/sync/push", auth(s.handleSyncPush))
	mux.HandleFunc("/api/sync/pull", auth(s.handleSyncPull))
	mux.HandleFunc("/api/sync/status", auth(s.handleSyncStatus))
	mux.HandleFunc("/api/vectors", auth(s.handleVectors))
	mux.HandleFunc("/api/vectors/build", auth(s.handleVectorsBuild))
}

// withAuth wraps a handler with PIN authentication if configured.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pin := s.cfg.Server.DashboardPIN
		if pin == "" {
			next(w, r)
			return
		}

		// Check token from header or cookie.
		token := r.Header.Get("X-Kai-Token")
		if token == "" {
			if c, err := r.Cookie("kai_session"); err == nil {
				token = c.Value
			}
		}
		if token != "" {
			if expiry, ok := s.sessions.Load(token); ok {
				if expiry.(int64) > time.Now().Unix() {
					next(w, r)
					return
				}
				s.sessions.Delete(token)
			}
		}

		// For API calls, return 401.
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.Contains(r.URL.Path, "/api/") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// For page requests, serve the lock screen.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		serveEmbedded(w, "templates/lock.html")
	}
}

// handleAuth handles PIN verification.
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		// GET: check if auth is required.
		writeJSON(w, map[string]bool{"required": s.cfg.Server.DashboardPIN != ""})
		return
	}

	var req struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	if req.PIN != s.cfg.Server.DashboardPIN {
		http.Error(w, `{"error":"invalid pin"}`, http.StatusUnauthorized)
		return
	}

	// Generate session token.
	token := generateToken()
	expiry := time.Now().Add(7 * 24 * time.Hour).Unix()
	s.sessions.Store(token, expiry)
	s.saveSessions()

	http.SetCookie(w, &http.Cookie{
		Name:     "kai_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	writeJSON(w, map[string]string{"status": "ok"})
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	serveEmbedded(w, "templates/index.html")
}

// serveEmbedded serves a raw embedded file as HTML.
func serveEmbedded(w http.ResponseWriter, name string) {
	data, err := templateFS.ReadFile(name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	type response struct {
		TotalFiles  int            `json:"total_files"`
		Categories  map[string]int `json:"categories"`
		TotalWords  int            `json:"total_words"`
		TotalChunks int            `json:"total_chunks"`
		IndexHealth string         `json:"index_health"`
		UniqueTags  int            `json:"unique_tags"`
		VecProvider string         `json:"vec_provider"`
		VecDims     int            `json:"vec_dims"`
		VecChunks   int            `json:"vec_chunks"`
		VecStatus   string         `json:"vec_status"`
	}

	resp := response{Categories: make(map[string]int)}

	idx, err := s.brain.LoadFullIndex()
	if err != nil {
		// Fall back to basic file list.
		files, _ := s.brain.ListAll()
		resp.TotalFiles = len(files)
		resp.IndexHealth = "missing"
		for _, f := range files {
			cat := strings.SplitN(f, "/", 2)[0]
			resp.Categories[cat]++
		}
		writeJSON(w, resp)
		return
	}

	resp.TotalFiles = idx.TotalDocs
	resp.TotalChunks = idx.TotalChunks

	if idx.TotalChunks > 0 {
		resp.IndexHealth = "ok"
	} else {
		resp.IndexHealth = "stale"
	}

	tags := make(map[string]bool)
	for _, doc := range idx.Documents {
		cat := strings.SplitN(doc.Path, "/", 2)[0]
		resp.Categories[cat]++
		resp.TotalWords += doc.WordCount
		for _, t := range doc.Tags {
			tags[t] = true
		}
	}
	resp.UniqueTags = len(tags)

	// Load vector index info.
	if vi, err := s.brain.LoadVecIndex(); err == nil && vi != nil {
		resp.VecProvider = vi.Provider
		resp.VecDims = vi.Dims
		resp.VecChunks = len(vi.ChunkVecs)
		resp.VecStatus = "ready"
	} else {
		resp.VecStatus = "none"
	}

	writeJSON(w, resp)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Items []taskItem `json:"items"`
	}

	// Find today's tasks file or the most recent one.
	tasksDir := filepath.Join(s.brainPath, "tasks")
	files, _ := filepath.Glob(filepath.Join(tasksDir, "*.md"))
	if len(files) == 0 {
		writeJSON(w, response{Items: []taskItem{}})
		return
	}

	// Sort descending to get the most recent date first.
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	bf, err := s.brain.Load("tasks/" + filepath.Base(files[0]))
	if err != nil {
		writeJSON(w, response{Items: []taskItem{}})
		return
	}

	items := parseTasks(bf.Content)
	writeJSON(w, response{Items: items})
}

var taskLineRe = regexp.MustCompile(`^- \[([ x~])\] (.+)$`)
var priorityRe = regexp.MustCompile(`\x{1f534}\s*high|\x{1f7e1}\s*medium|\x{1f7e2}\s*low`)

func parseTasks(content string) []taskItem {
	var items []taskItem
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		m := taskLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		done := m[1] == "x"
		text := m[2]

		// Extract priority from emoji markers.
		var priority string
		if strings.Contains(text, "high") {
			priority = "high"
		} else if strings.Contains(text, "medium") {
			priority = "medium"
		} else if strings.Contains(text, "low") {
			priority = "low"
		}

		items = append(items, taskItem{Text: text, Done: done, Priority: priority})
	}
	return items
}

type taskItem struct {
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Priority string `json:"priority,omitempty"`
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	type activityEntry struct {
		Path        string `json:"path"`
		WordCount   int    `json:"word_count"`
		AccessCount int    `json:"access_count"`
		Updated     string `json:"updated"`
	}
	type response struct {
		Recent []activityEntry `json:"recent"`
	}

	idx, err := s.brain.LoadFullIndex()
	if err != nil {
		writeJSON(w, response{Recent: []activityEntry{}})
		return
	}

	entries := make([]activityEntry, 0, len(idx.Documents))
	for _, doc := range idx.Documents {
		entries = append(entries, activityEntry{
			Path:        doc.Path,
			WordCount:   doc.WordCount,
			AccessCount: doc.AccessCount,
			Updated:     doc.Updated,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Updated == entries[j].Updated {
			return entries[i].AccessCount > entries[j].AccessCount
		}
		return entries[i].Updated > entries[j].Updated
	})

	if len(entries) > 20 {
		entries = entries[:20]
	}

	writeJSON(w, response{Recent: entries})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	type searchResult struct {
		DocPath string  `json:"doc_path"`
		ChunkID int     `json:"chunk_id"`
		Score   float64 `json:"score"`
		Snippet string  `json:"snippet"`
	}
	type response struct {
		Query   string         `json:"query"`
		Results []searchResult `json:"results"`
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, response{Query: q, Results: []searchResult{}})
		return
	}

	chunks := s.brain.QueryWithPRF(q)

	var results []searchResult
	seen := make(map[string]int) // doc path → chunk count
	for _, cr := range chunks {
		if seen[cr.DocPath] >= 2 {
			continue // max 2 chunks per file
		}
		if len(results) >= 20 {
			break
		}

		snippet := ""
		bf, err := s.brain.Load(cr.DocPath)
		if err == nil {
			content := strings.TrimSpace(bf.Content)
			snippet = extractSnippet(content, cr.Offset, cr.Length, 200)
		}

		results = append(results, searchResult{
			DocPath: cr.DocPath,
			ChunkID: cr.ChunkID,
			Score:   cr.Score,
			Snippet: snippet,
		})
		seen[cr.DocPath]++
	}

	writeJSON(w, response{Query: q, Results: results})
}

func extractSnippet(content string, offset, length, maxLen int) string {
	if offset >= len(content) {
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}
	end := offset + length
	if end > len(content) {
		end = len(content)
	}
	s := strings.TrimSpace(content[offset:end])
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	type response struct {
		AgentName string `json:"agent_name"`
		UserName  string `json:"user_name"`
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildDate string `json:"build_date"`
	}

	writeJSON(w, response{
		AgentName: s.cfg.AgentName(),
		UserName:  s.cfg.UserName(),
		Version:   kai.Version,
		Commit:    kai.Commit,
		BuildDate: kai.BuildDate,
	})
}

func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	type tagEntry struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	type response struct {
		Tags []tagEntry `json:"tags"`
	}

	idx, err := s.brain.LoadFullIndex()
	if err != nil {
		writeJSON(w, response{Tags: []tagEntry{}})
		return
	}

	counts := make(map[string]int)
	for _, doc := range idx.Documents {
		for _, t := range doc.Tags {
			counts[t]++
		}
	}

	tags := make([]tagEntry, 0, len(counts))
	for name, count := range counts {
		tags = append(tags, tagEntry{Name: name, Count: count})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Count > tags[j].Count })

	if len(tags) > 50 {
		tags = tags[:50]
	}

	writeJSON(w, response{Tags: tags})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	type fileEntry struct {
		Path        string   `json:"path"`
		Category    string   `json:"category"`
		Tags        []string `json:"tags"`
		Confidence  string   `json:"confidence"`
		WordCount   int      `json:"word_count"`
		AccessCount int      `json:"access_count"`
		Updated     string   `json:"updated"`
	}
	type response struct {
		Files []fileEntry `json:"files"`
	}

	files, err := s.brain.ListAll()
	if err != nil {
		writeJSON(w, response{Files: []fileEntry{}})
		return
	}

	idx, _ := s.brain.LoadFullIndex()

	entries := make([]fileEntry, 0, len(files))
	for _, f := range files {
		cat := strings.SplitN(f, "/", 2)[0]
		entry := fileEntry{Path: f, Category: cat}

		if idx != nil {
			if doc, ok := idx.Documents[f]; ok {
				entry.Tags = doc.Tags
				entry.WordCount = doc.WordCount
				entry.AccessCount = doc.AccessCount
				entry.Updated = doc.Updated
			}
		}

		bf, err := s.brain.Load(f)
		if err == nil {
			entry.Confidence = bf.Confidence
			if entry.Tags == nil {
				entry.Tags = bf.Tags
			}
		}
		if entry.Tags == nil {
			entry.Tags = []string{}
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	writeJSON(w, response{Files: entries})
}

func (s *Server) handleFileContent(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Path       string   `json:"path"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
		Content    string   `json:"content"`
		Updated    string   `json:"updated"`
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path parameter", http.StatusBadRequest)
		return
	}

	bf, err := s.brain.Load(path)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	writeJSON(w, response{
		Path:       bf.Path,
		Tags:       bf.Tags,
		Confidence: bf.Confidence,
		Content:    strings.TrimSpace(bf.Content),
		Updated:    bf.Updated.Time.Format(brain.DateFormat),
	})
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form for file uploads
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		writeJSON(w, map[string]interface{}{"error": "failed to parse upload"})
		return
	}

	category := r.FormValue("category")
	if category == "" {
		category = "knowledge"
	}
	tagsStr := r.FormValue("tags")
	confidence := r.FormValue("confidence")
	if confidence == "" {
		confidence = "medium"
	}
	autoTag := r.FormValue("autoTag") == "true"
	autoChunk := r.FormValue("autoChunk") == "true"

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}

	type importResult struct {
		Path  string `json:"path"`
		Error string `json:"error,omitempty"`
	}
	var results []importResult

	for _, fileHeader := range r.MultipartForm.File["files"] {
		file, err := fileHeader.Open()
		if err != nil {
			results = append(results, importResult{
				Path:  fileHeader.Filename,
				Error: "failed to open file",
			})
			continue
		}
		defer file.Close()

		data := make([]byte, fileHeader.Size)
		if _, err := file.Read(data); err != nil {
			results = append(results, importResult{
				Path:  fileHeader.Filename,
				Error: "failed to read file",
			})
			continue
		}

		bf, err := importFileData(fileHeader.Filename, data, category, tags, confidence, autoTag, autoChunk)
		if err != nil {
			results = append(results, importResult{
				Path:  fileHeader.Filename,
				Error: err.Error(),
			})
			continue
		}

		if err := s.brain.Save(bf); err != nil {
			results = append(results, importResult{
				Path:  fileHeader.Filename,
				Error: "failed to save: " + err.Error(),
			})
			continue
		}

		results = append(results, importResult{Path: bf.Path})
	}

	// Rebuild indexes
	if len(results) > 0 {
		_ = s.brain.RebuildIndex()
	}

	writeJSON(w, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func (s *Server) handleReorg(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Actions []brain.ReorgAction `json:"actions"`
		Count   int                 `json:"count"`
	}

	opts := brain.ReorgOptions{
		Mode:                brain.ModeAll,
		SimilarityThreshold: 0.7,
		SmallFileThreshold:  50,
		DryRun:              true,
	}

	plan, err := s.brain.Reorganize(opts)
	if err != nil || plan == nil {
		writeJSON(w, response{Actions: []brain.ReorgAction{}, Count: 0})
		return
	}

	actions := plan.Actions
	if actions == nil {
		actions = []brain.ReorgAction{}
	}

	writeJSON(w, response{Actions: actions, Count: len(actions)})
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	tools := s.getToolDefs()
	writeJSON(w, map[string]interface{}{"tools": tools})
}

func (s *Server) handleToolExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Tool   string            `json:"tool"`
		Params map[string]string `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"error": "bad request"})
		return
	}

	result, err := s.executeTool(req.Tool, req.Params)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"result": result})
}

// getToolDefs returns the list of available tools with their parameters.
func (s *Server) getToolDefs() []ToolDef {
	return []ToolDef{
		{Name: "brain_list", Description: "List all brain knowledge files", Params: nil},
		{Name: "brain_read", Description: "Read a brain file", Params: []ToolParam{
			{Name: "path", Type: "string", Required: true, Description: "File path (e.g. opinions/testing.md)"},
		}},
		{Name: "brain_search", Description: "Search brain by keywords", Params: []ToolParam{
			{Name: "query", Type: "string", Required: true, Description: "Search keywords"},
		}},
		{Name: "brain_write", Description: "Write a brain file", Params: []ToolParam{
			{Name: "path", Type: "string", Required: true, Description: "File path"},
			{Name: "content", Type: "string", Required: true, Description: "Content to write"},
			{Name: "tags", Type: "string", Required: true, Description: "Comma-separated tags"},
			{Name: "confidence", Type: "string", Required: false, Description: "high, medium, or low"},
		}},
		{Name: "ask", Description: "Ask a question using brain context", Params: []ToolParam{
			{Name: "question", Type: "string", Required: true, Description: "The question to ask"},
		}},
		{Name: "brain_reorganize", Description: "Analyze brain for reorganization", Params: []ToolParam{
			{Name: "mode", Type: "string", Required: false, Description: "all, dedup, recategorize, or consolidate"},
			{Name: "apply", Type: "boolean", Required: false, Description: "Execute plan (default: false)"},
		}},
	}
}

// executeTool runs a tool and returns the result as a string.
func (s *Server) executeTool(toolName string, params map[string]string) (string, error) {
	switch toolName {
	case "brain_list":
		files, err := s.brain.ListAll()
		if err != nil {
			return "", err
		}
		data, _ := json.MarshalIndent(files, "", "  ")
		return string(data), nil

	case "brain_read":
		path := params["path"]
		if path == "" {
			return "", fmt.Errorf("path is required")
		}
		bf, err := s.brain.Load(path)
		if err != nil {
			return "", err
		}
		type readResult struct {
			Path       string   `json:"path"`
			Tags       []string `json:"tags"`
			Confidence string   `json:"confidence"`
			Content    string   `json:"content"`
		}
		data, _ := json.MarshalIndent(readResult{
			Path: bf.Path, Tags: bf.Tags, Confidence: bf.Confidence,
			Content: strings.TrimSpace(bf.Content),
		}, "", "  ")
		return string(data), nil

	case "brain_search":
		query := params["query"]
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		chunks := s.brain.QueryHybrid(query)
		type searchResult struct {
			Path    string  `json:"path"`
			ChunkID int     `json:"chunk_id"`
			Score   float64 `json:"score"`
		}
		var results []searchResult
		for i, cr := range chunks {
			if i >= 10 {
				break
			}
			results = append(results, searchResult{Path: cr.DocPath, ChunkID: cr.ChunkID, Score: cr.Score})
		}
		data, _ := json.MarshalIndent(results, "", "  ")
		return string(data), nil

	case "brain_write":
		path := params["path"]
		content := params["content"]
		tags := params["tags"]
		if path == "" || content == "" || tags == "" {
			return "", fmt.Errorf("path, content, and tags are required")
		}
		confidence := params["confidence"]
		if confidence == "" {
			confidence = "medium"
		}
		tagList := strings.Split(tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
		bf := &brain.BrainFile{
			Path:       path,
			Tags:       tagList,
			Confidence: confidence,
			Source:     "dashboard",
			Content:    "\n" + content + "\n",
		}
		if err := s.brain.Save(bf); err != nil {
			return "", err
		}
		if err := s.brain.RebuildIndex(); err != nil {
			return "", fmt.Errorf("saved but index rebuild failed: %w", err)
		}
		return fmt.Sprintf("Saved %s", path), nil

	case "ask":
		question := params["question"]
		if question == "" {
			return "", fmt.Errorf("question is required")
		}
		chunks := s.brain.QueryHybrid(question)
		var sb strings.Builder
		seen := make(map[string]bool)
		count := 0
		for _, cr := range chunks {
			if count >= 3 || seen[cr.DocPath] {
				continue
			}
			seen[cr.DocPath] = true
			bf, err := s.brain.Load(cr.DocPath)
			if err != nil {
				continue
			}
			content := strings.TrimSpace(bf.Content)
			if cr.Offset < len(content) {
				end := cr.Offset + cr.Length
				if end > len(content) {
					end = len(content)
				}
				content = strings.TrimSpace(content[cr.Offset:end])
			}
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString("## ")
			sb.WriteString(cr.DocPath)
			sb.WriteString("\n")
			sb.WriteString(content)
			count++
		}
		if sb.Len() == 0 {
			return "No relevant brain files found.", nil
		}
		return sb.String(), nil

	case "brain_reorganize":
		opts := brain.DefaultReorgOptions()
		opts.DryRun = params["apply"] != "true"
		if m := params["mode"]; m != "" {
			opts.Mode = m
		}
		plan, err := s.brain.Reorganize(opts)
		if err != nil {
			return "", err
		}
		if plan == nil || len(plan.Actions) == 0 {
			return "No reorganization needed.", nil
		}
		data, _ := json.MarshalIndent(plan.Actions, "", "  ")
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		Mode    string `json:"mode"` // "brain" (default) or "free"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, map[string]string{"error": "message is required"})
		return
	}
	if req.Mode == "" {
		req.Mode = "brain"
	}

	// Free chat: direct LLM call, no brain context.
	if req.Mode == "free" {
		if s.router == nil {
			writeJSON(w, map[string]string{"error": "no LLM provider configured"})
			return
		}
		result, err := s.router.FreeChat(r.Context(), req.Message)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{"response": result.Response})
		return
	}

	// Brain chat: retrieve context + LLM.
	if s.router != nil {
		result, err := s.router.Ask(r.Context(), req.Message)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{
			"response":         result.Response,
			"confidence":       result.Confidence,
			"brain_files_used": result.BrainFilesUsed,
		})
		return
	}

	// Without LLM: return brain search results as context.
	chunks := s.brain.QueryHybrid(req.Message)
	var sb strings.Builder
	seen := make(map[string]bool)
	count := 0
	for _, cr := range chunks {
		if count >= 3 || seen[cr.DocPath] {
			continue
		}
		seen[cr.DocPath] = true
		bf, err := s.brain.Load(cr.DocPath)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(bf.Content)
		if cr.Offset < len(content) {
			end := cr.Offset + cr.Length
			if end > len(content) {
				end = len(content)
			}
			content = strings.TrimSpace(content[cr.Offset:end])
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("## ")
		sb.WriteString(cr.DocPath)
		sb.WriteString("\n")
		sb.WriteString(content)
		count++
	}
	if sb.Len() == 0 {
		writeJSON(w, map[string]string{"response": "No relevant brain files found."})
		return
	}
	writeJSON(w, map[string]interface{}{
		"response":   sb.String(),
		"confidence": "context_only",
	})
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		Mode    string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = "brain"
	}

	if s.router == nil {
		http.Error(w, `{"error":"no LLM provider configured"}`, http.StatusServiceUnavailable)
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	var brainPaths []string

	if req.Mode == "free" {
		ch, err := s.router.StreamFreeChat(r.Context(), req.Message)
		if err != nil {
			fmt.Fprintf(w, "data: {\"error\":%q}\n\n", err.Error())
			flusher.Flush()
			return
		}
		for chunk := range ch {
			if chunk.Done {
				break
			}
			data, _ := json.Marshal(map[string]string{"content": chunk.Content})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	} else {
		ch, paths, err := s.router.StreamAsk(r.Context(), req.Message)
		if err != nil {
			fmt.Fprintf(w, "data: {\"error\":%q}\n\n", err.Error())
			flusher.Flush()
			return
		}
		brainPaths = paths
		for chunk := range ch {
			if chunk.Done {
				break
			}
			data, _ := json.Marshal(map[string]string{"content": chunk.Content})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}

	// Send final event with metadata.
	done, _ := json.Marshal(map[string]interface{}{
		"done":             true,
		"brain_files_used": brainPaths,
	})
	fmt.Fprintf(w, "data: %s\n\n", done)
	flusher.Flush()
}

func (s *Server) handleSyncPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.syncer == nil || !s.syncer.HasBackends() {
		writeJSON(w, map[string]string{"error": "no sync backends configured"})
		return
	}
	if err := s.syncer.Push(r.Context(), false, false); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": "pushed"})
}

func (s *Server) handleSyncPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.syncer == nil || !s.syncer.HasBackends() {
		writeJSON(w, map[string]string{"error": "no sync backends configured"})
		return
	}
	if err := s.syncer.Pull(r.Context(), false, false); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	// Rebuild index after pull.
	if err := s.brain.RebuildIndex(); err != nil {
		writeJSON(w, map[string]string{"status": "pulled", "warning": "index rebuild failed: " + err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": "pulled"})
}

func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if s.syncer == nil || !s.syncer.HasBackends() {
		writeJSON(w, map[string]interface{}{"configured": false})
		return
	}
	gitStatus, cloudStatus, err := s.syncer.StatusAll(r.Context())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	resp := map[string]interface{}{"configured": true}
	if gitStatus != nil {
		resp["git"] = gitStatus
	}
	if cloudStatus != nil {
		resp["cloud"] = cloudStatus
	}
	writeJSON(w, resp)
}

// extractRepoName extracts the bare repo name from a git URL
// Examples: "git@github.com:user/kai-brain.git" -> "kai-brain"
//          "https://github.com/user/kai-brain.git" -> "kai-brain"
func extractRepoName(gitURL string) string {
	// Remove .git suffix
	name := strings.TrimSuffix(gitURL, ".git")
	// Get the last path segment
	parts := strings.Split(name, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Services []serviceStatus `json:"services"`
	}

	var services []serviceStatus

	// 1. LLM Provider status
	provName := s.cfg.Provider
	if provName == "" {
		services = append(services, serviceStatus{
			Name: "LLM Provider", Status: "not_configured", Value: "—", Tooltip: "No provider configured",
		})
	} else {
		provCfg, ok := s.cfg.Providers[provName]
		if !ok {
			services = append(services, serviceStatus{
				Name: "LLM Provider", Status: "disconnected", Value: provName, Tooltip: "Provider not found in config",
			})
		} else {
			tooltip := provCfg.Model
			if provCfg.BaseURL != "" {
				tooltip += " @ " + provCfg.BaseURL
			}
			if provCfg.APIKey == "" {
				tooltip += " (no API key)"
			}
			status := "connected"
			if provCfg.APIKey == "" {
				status = "disconnected"
			}
			services = append(services, serviceStatus{
				Name: "LLM Provider", Status: status, Value: provName, Tooltip: tooltip,
			})
		}
	}

	// 2. Embeddings status
	embedProvider := s.cfg.Brain.Embeddings.Provider
	if embedProvider == "" {
		embedProvider = "auto"
	}

	vi, viErr := s.brain.LoadVecIndex()
	if viErr != nil {
		// No vector index yet — check what's configured
		if embedProvider == "ollama" || embedProvider == "auto" {
			ollamaURL := s.cfg.Brain.Embeddings.Ollama.URL
			if ollamaURL == "" {
				ollamaURL = brain.DefaultOllamaURL
			}
			model := s.cfg.Brain.Embeddings.Ollama.Model
			if model == "" {
				model = brain.DefaultEmbedModel
			}
			embedder := brain.NewOllamaEmbedder(brain.OllamaEmbedConfig{
				URL: ollamaURL, Model: model,
			})
			if embedder.Available() {
				services = append(services, serviceStatus{
					Name: "Embeddings", Status: "connected",
					Value: "Ollama", Tooltip: model + " @ " + ollamaURL + " (index not built)",
				})
			} else if embedProvider == "auto" {
				services = append(services, serviceStatus{
					Name: "Embeddings", Status: "connected", Value: "LSI", Tooltip: "No vector index yet",
				})
			} else {
				services = append(services, serviceStatus{
					Name: "Embeddings", Status: "disconnected",
					Value: "Ollama", Tooltip: model + " @ " + ollamaURL + " (unreachable)",
				})
			}
		} else {
			services = append(services, serviceStatus{
				Name: "Embeddings", Status: "connected", Value: "LSI", Tooltip: "No vector index yet",
			})
		}
	} else {
		tooltip := fmt.Sprintf("%s: %d dims, %d chunks", vi.Provider, vi.Dims, len(vi.ChunkVecs))
		services = append(services, serviceStatus{
			Name: "Embeddings", Status: "connected", Value: vi.Provider, Tooltip: tooltip,
		})
	}

	// 3. Brain Sync status
	if s.cfg.Brain.Sync.Git.Enabled {
		repoName := "git"
		tooltip := "Git sync enabled"
		if s.cfg.Brain.Sync.Git.Remote != "" {
			repoName = extractRepoName(s.cfg.Brain.Sync.Git.Remote)
			tooltip = s.cfg.Brain.Sync.Git.Remote
		}
		services = append(services, serviceStatus{
			Name: "Brain Sync", Status: "connected", Value: repoName, Tooltip: tooltip,
		})
	} else {
		services = append(services, serviceStatus{
			Name: "Brain Sync", Status: "not_configured", Value: "—", Tooltip: "Git sync not enabled",
		})
	}

	writeJSON(w, response{Services: services})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func importFileData(filename string, data []byte, category string, tags []string, confidence string, autoTag bool, autoChunk bool) (*brain.BrainFile, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	baseFileName := strings.TrimSuffix(filename, ext)

	var content string
	switch ext {
	case ".md":
		content = string(data)

	case ".csv":
		// Parse CSV and convert to markdown table
		records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
		if err != nil {
			return nil, fmt.Errorf("parse CSV: %w", err)
		}

		if len(records) == 0 {
			return nil, fmt.Errorf("CSV is empty")
		}

		// Build markdown table with proper escaping
		content = brain.BuildMarkdownTable(records)

	case ".xlsx":
		// Parse Excel file and convert to markdown tables
		f, err := excelize.OpenReader(io.NopCloser(strings.NewReader(string(data))))
		if err != nil {
			return nil, fmt.Errorf("open Excel file: %w", err)
		}
		defer f.Close()

		sheetNames := f.GetSheetList()
		if len(sheetNames) == 0 {
			return nil, fmt.Errorf("Excel file has no sheets")
		}

		// Get all rows from all sheets
		var buf strings.Builder
		for i, sheetName := range sheetNames {
			if i > 0 {
				buf.WriteString("\n\n")
			}
			if len(sheetNames) > 1 {
				buf.WriteString("## " + sheetName + "\n\n")
			}

			rows, err := f.GetRows(sheetName)
			if err != nil {
				return nil, fmt.Errorf("read sheet %s: %w", sheetName, err)
			}

			if len(rows) == 0 {
				continue
			}

			// Build markdown table with proper escaping
			if len(rows[0]) > 0 {
				buf.WriteString(brain.BuildMarkdownTable(rows))
			}
		}
		content = buf.String()

	default:
		return nil, fmt.Errorf("unsupported file type: %s (only .md, .csv, and .xlsx supported)", ext)
	}

	// Construct brain file path
	brainPath := filepath.Join(category, baseFileName+".md")
	brainPath = filepath.Clean(strings.ReplaceAll(brainPath, "\\", "/"))

	// Auto-extract tags if enabled
	finalTags := tags
	if autoTag {
		extractedTags := brain.ExtractTags(content)
		// Merge extracted tags with provided tags
		seen := make(map[string]bool)
		for _, t := range tags {
			seen[t] = true
		}
		for _, t := range extractedTags {
			if !seen[t] {
				finalTags = append(finalTags, t)
				seen[t] = true
			}
		}
	}

	return &brain.BrainFile{
		Path:       brainPath,
		Content:    "\n" + strings.TrimSpace(content) + "\n",
		Tags:       finalTags,
		Confidence: confidence,
		Source:     "imported:" + filename,
		Updated:    brain.DateOnly{Time: time.Now()},
	}, nil
}
