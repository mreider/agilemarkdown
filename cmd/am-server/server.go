package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mreider/agilemarkdown/mcpserver"
)

// staticFS holds the embedded web bundle. The webpack web target
// writes its output to vscode/dist/web/, which the repo's build
// step copies into cmd/am-server/web/ before `go build`.
//
//go:embed all:web
var staticFS embed.FS

type server struct {
	root string

	// snapshot fan-out: a single watcher pushes the freshest snapshot
	// to every connected /ws/snapshot client. Centralized so the
	// HTTP write routes can also poke the watcher (the underlying
	// fsnotify event will follow, but the explicit poke avoids the
	// 200ms debounce delay on user-driven mutations).
	subsMu sync.Mutex
	subs   map[chan []byte]struct{}

	// log is the CommandLog feed every write route publishes to.
	// Subscribers connect on /ws/log.
	log *logBus

	// mcp is the single shared `am mcp` child process. Lazy: we
	// start it on first /api/mcp request so a missing `am` binary
	// doesn't kill the lab altogether.
	mcpMu sync.Mutex
	mcp   *mcpBridge

	watcher *watcher
}

func newServer(root string) *server {
	s := &server{
		root: root,
		subs: make(map[chan []byte]struct{}),
		log:  newLogBus(),
	}
	w, err := newWatcher(root, func(snap []byte) {
		s.broadcast(snap)
	})
	if err != nil {
		log.Printf("watcher disabled: %v", err)
	} else {
		s.watcher = w
	}
	return s
}

func (s *server) Close() {
	if s.watcher != nil {
		s.watcher.Close()
	}
	if s.mcp != nil {
		s.mcp.Close()
	}
}

// ensureMcp lazily spawns the single shared `am mcp` child. Returns
// nil bridge + nil error in the degenerate case so callers can check
// b == nil cleanly.
func (s *server) ensureMcp() (*mcpBridge, error) {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	if s.mcp != nil {
		return s.mcp, nil
	}
	b, err := newMcpBridge(s.root)
	if err != nil {
		return nil, err
	}
	s.mcp = b
	return b, nil
}

func (s *server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Read endpoints.
	mux.HandleFunc("GET /api/snapshot", s.handleSnapshot)
	mux.HandleFunc("GET /api/get-item", s.handleGetItem)

	// Write endpoints.
	mux.HandleFunc("POST /api/set-status", s.handleSetStatus)
	s.registerWriteRoutes(mux)

	// MCP console.
	mux.HandleFunc("GET /api/mcp-tools", s.handleMcpTools)
	mux.HandleFunc("POST /api/mcp/{tool}", s.handleMcpCall)

	// WebSocket: snapshot stream, log feed, terminal pty.
	mux.HandleFunc("/ws/snapshot", s.handleWsSnapshot)
	mux.HandleFunc("/ws/log", s.handleWsLog)
	mux.HandleFunc("/ws/pty", s.handleWsPty)

	// Static bundle from the embedded FS, rooted under web/.
	sub, _ := fs.Sub(staticFS, "web")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	return mux
}

func (s *server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := buildSnapshot(s.root)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

type setStatusReq struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

func (s *server) handleSetStatus(w http.ResponseWriter, r *http.Request) {
	var req setStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if req.Path == "" || req.Status == "" {
		http.Error(w, "path and status required", 400)
		return
	}
	verb := "am " + req.Status + " " + req.Path
	c := s.call(verb)
	if err := callSetStatus(s.root, req.Path, req.Status); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok(summarizeItem(s.root, req.Path))
	s.pokeSnapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// pokeSnapshot forces a fresh snapshot push out of band with the
// watcher debounce. UI mutations should reflect immediately rather
// than wait for the fsnotify event to land.
func (s *server) pokeSnapshot() {
	snap, err := buildSnapshot(s.root)
	if err != nil {
		log.Printf("poke build: %v", err)
		return
	}
	payload, _ := json.Marshal(snap)
	s.broadcast(payload)
}

func (s *server) broadcast(payload []byte) {
	s.subsMu.Lock()
	subs := make([]chan []byte, 0, len(s.subs))
	for c := range s.subs {
		subs = append(subs, c)
	}
	s.subsMu.Unlock()
	for _, c := range subs {
		select {
		case c <- payload:
		default:
			// Slow consumer; drop. Next push will catch them up.
		}
	}
}

// handleGetItem returns the same payload the extension's
// `open-detail` message synthesizes: full frontmatter + body + the
// item's comments + tasks. The lab's web-api shim composes a
// `detail` MessageEvent from it so the DetailPanel renders the
// same way it does in the extension.
func (s *server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", 400)
		return
	}
	item, err := mcpserver.GetItem(r.Context(), s.root, mcpserver.GetItemArgs{Path: path})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	comments, _ := mcpserver.GetComments(r.Context(), s.root, mcpserver.GetCommentsArgs{Path: path})
	tasks, _ := mcpserver.ListTasks(r.Context(), s.root, mcpserver.ListTasksArgs{Path: path})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"item":     item,
		"comments": comments,
		"tasks":    tasks,
	})
}

// ---- MCP console -----------------------------------------------------------

func (s *server) handleMcpTools(w http.ResponseWriter, r *http.Request) {
	bridge, err := s.ensureMcp()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	res, err := bridge.ListTools(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func (s *server) handleMcpCall(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool")
	if tool == "" {
		http.Error(w, "tool name required", 400)
		return
	}
	var args any
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	bridge, err := s.ensureMcp()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	trace, err := bridge.CallTool(r.Context(), tool, args)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Emit a CommandLog entry so the MCP call also appears in the
	// terminal feed, pedagogically useful: the same action shows up
	// in two places at once.
	s.log.emit(fmt.Sprintf("mcp tools/call %s", tool), true, "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}

// resolveItemPath turns a posted path like "stories/foo.md" into an
// absolute path inside the workspace, refusing anything that escapes
// the root.
func (s *server) resolveItemPath(rel string) (string, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	abs := filepath.Join(s.root, rel)
	if !strings.HasPrefix(abs, s.root) {
		return "", false
	}
	return abs, true
}
