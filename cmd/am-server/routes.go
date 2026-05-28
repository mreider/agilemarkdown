package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mreider/agilemarkdown/mcpserver"
)

// route registers all HTTP write endpoints on the server's mux. Each
// route maps to one mcpserver.* wrapper, emits a CommandLog entry,
// and pokes the snapshot bus so connected clients see the change
// without waiting on the file-watcher debounce.
//
// The pattern in this file is the same for every route:
//   1. Decode JSON body into the matching Args struct.
//   2. Call the mcpserver wrapper (or an in-process helper).
//   3. Emit a CommandLog event describing the equivalent CLI verb.
//   4. Poke the snapshot bus and return {"ok": true}.
func (s *server) registerWriteRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/set-estimate", s.handleSetEstimate)
	mux.HandleFunc("POST /api/set-assigned", s.handleSetAssigned)
	mux.HandleFunc("POST /api/set-tags", s.handleSetTags)
	mux.HandleFunc("POST /api/set-epic", s.handleSetEpic)
	mux.HandleFunc("POST /api/set-type", s.handleSetType)
	mux.HandleFunc("POST /api/set-release-date", s.handleSetReleaseDate)
	mux.HandleFunc("POST /api/set-description", s.handleSetDescription)
	mux.HandleFunc("POST /api/rank", s.handleRank)
	mux.HandleFunc("POST /api/ice", s.handleIce)
	mux.HandleFunc("POST /api/unice", s.handleUnice)
	mux.HandleFunc("POST /api/move-to-icebox", s.handleMoveToIcebox)
	mux.HandleFunc("POST /api/move-to-priority", s.handleMoveToPriority)
	mux.HandleFunc("POST /api/comment", s.handleComment)
	mux.HandleFunc("POST /api/task", s.handleTask)
	mux.HandleFunc("POST /api/create-item", s.handleCreateItem)
	mux.HandleFunc("POST /api/delete-item", s.handleDeleteItem)
	mux.HandleFunc("POST /api/block", s.handleBlock)
	mux.HandleFunc("POST /api/unblock", s.handleUnblock)
	mux.HandleFunc("POST /api/reject", s.handleReject)
	mux.HandleFunc("POST /api/sync", s.handleSync)
}

// decode is a tiny helper: parses the request body or writes a 400.
func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		http.Error(w, "bad json: "+err.Error(), 400)
		return false
	}
	return true
}

func writeOK(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	if body == nil {
		body = map[string]any{"ok": true}
	}
	json.NewEncoder(w).Encode(body)
}

// quote wraps a value for the rendered verb. Skips quoting when the
// value has no shell-significant chars.
func quote(s string) string {
	if s == "" {
		return `""`
	}
	for _, c := range s {
		if c == ' ' || c == '"' || c == '\'' || c == '$' || c == '|' || c == ';' {
			return fmt.Sprintf("%q", s)
		}
	}
	return s
}

// ---- set-estimate -----------------------------------------------------------

func (s *server) handleSetEstimate(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.SetEstimateArgs
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am estimate %s %s", req.Path, quote(req.Estimate))
	c := s.call(verb)
	if _, err := mcpserver.SetEstimate(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok(summarizeItem(s.root, req.Path))
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- set-assigned -----------------------------------------------------------

func (s *server) handleSetAssigned(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.SetAssignedArgs
	if !decode(w, r, &req) {
		return
	}
	users := req.Assigned
	if len(req.Assignees) > 0 {
		users = ""
		for i, a := range req.Assignees {
			if i > 0 {
				users += ","
			}
			users += a
		}
	}
	verb := fmt.Sprintf("am assign %s %s", req.Path, quote(users))
	c := s.call(verb)
	if _, err := mcpserver.SetAssigned(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("assigned: " + users)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- set-tags ---------------------------------------------------------------

func (s *server) handleSetTags(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.SetTagsArgs
	if !decode(w, r, &req) {
		return
	}
	joined := ""
	for i, t := range req.Tags {
		if i > 0 {
			joined += ","
		}
		joined += t
	}
	verb := fmt.Sprintf("am tag %s %s", req.Path, quote(joined))
	c := s.call(verb)
	if _, err := mcpserver.SetTags(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("tags: " + joined)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- set-epic ---------------------------------------------------------------

func (s *server) handleSetEpic(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.SetEpicArgs
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am epic %s %s", req.Path, quote(req.Slug))
	c := s.call(verb)
	if _, err := mcpserver.SetEpic(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("epic: " + req.Slug)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- set-type ---------------------------------------------------------------

type setTypeReq struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func (s *server) handleSetType(w http.ResponseWriter, r *http.Request) {
	var req setTypeReq
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am set-type %s %s", req.Path, req.Type)
	c := s.call(verb)
	if err := callSetType(s.root, req.Path, req.Type); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("type: " + req.Type)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- set-release-date -------------------------------------------------------

type setReleaseDateReq struct {
	Path        string `json:"path"`
	ReleaseDate string `json:"release_date"`
}

func (s *server) handleSetReleaseDate(w http.ResponseWriter, r *http.Request) {
	var req setReleaseDateReq
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am set-release-date %s %s", req.Path, req.ReleaseDate)
	c := s.call(verb)
	if err := callSetReleaseDate(s.root, req.Path, req.ReleaseDate); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("release_date: " + req.ReleaseDate)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- set-description --------------------------------------------------------

func (s *server) handleSetDescription(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.SetDescriptionArgs
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am set-description %s < (stdin)", req.Path)
	c := s.call(verb)
	if _, err := mcpserver.SetDescription(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("description updated")
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- rank -------------------------------------------------------------------

func (s *server) handleRank(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.RankItemArgs
	if !decode(w, r, &req) {
		return
	}
	flag := req.Position
	if flag == "" && req.After != "" {
		flag = "--after " + req.After
	} else if flag == "" && req.Before != "" {
		flag = "--before " + req.Before
	} else if flag != "" {
		flag = "--" + flag
	}
	verb := fmt.Sprintf("am rank %s %s", req.ItemPath, flag)
	c := s.call(verb)
	if _, err := mcpserver.RankItem(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("ranked " + req.ItemPath)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- ice / unice ------------------------------------------------------------

type pathReq struct {
	Backlog  string `json:"backlog"`
	ItemPath string `json:"item_path"`
}

func (s *server) handleIce(w http.ResponseWriter, r *http.Request) {
	var req pathReq
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am ice %s", req.ItemPath)
	c := s.call(verb)
	if _, err := mcpserver.MoveToIcebox(context.Background(), s.root, mcpserver.MoveToIceboxArgs{
		Backlog: req.Backlog, ItemPath: req.ItemPath,
	}); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("moved to icebox")
	s.pokeSnapshot()
	writeOK(w, nil)
}

func (s *server) handleUnice(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.MoveToPriorityArgs
	if !decode(w, r, &req) {
		return
	}
	verb := "am unice"
	if len(req.ItemPaths) > 0 {
		verb = "am unice " + req.ItemPaths[0]
	}
	c := s.call(verb)
	if _, err := mcpserver.MoveToPriority(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("moved to priority")
	s.pokeSnapshot()
	writeOK(w, nil)
}

func (s *server) handleMoveToIcebox(w http.ResponseWriter, r *http.Request) { s.handleIce(w, r) }
func (s *server) handleMoveToPriority(w http.ResponseWriter, r *http.Request) {
	s.handleUnice(w, r)
}

// ---- comment ---------------------------------------------------------------

func (s *server) handleComment(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.AddCommentArgs
	if !decode(w, r, &req) {
		return
	}
	author := req.Author
	if author == "" {
		author = "lab"
	}
	verb := fmt.Sprintf("am comment %s %s --author %s", req.Path, quote(req.Text), author)
	c := s.call(verb)
	if _, err := mcpserver.AddComment(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("comment added")
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- task add / tick --------------------------------------------------------

type taskReq struct {
	Path string `json:"path"`
	Text string `json:"text,omitempty"`
	Idx  *int   `json:"idx,omitempty"`
}

func (s *server) handleTask(w http.ResponseWriter, r *http.Request) {
	var req taskReq
	if !decode(w, r, &req) {
		return
	}
	if req.Idx != nil {
		verb := fmt.Sprintf("am task tick %s %d", req.Path, *req.Idx)
		c := s.call(verb)
		if _, err := mcpserver.SetTaskDone(context.Background(), s.root, mcpserver.SetTaskDoneArgs{
			Path: req.Path, Index: *req.Idx,
		}); err != nil {
			http.Error(w, err.Error(), 400)
			c.fail(err)
			return
		}
		c.ok("task toggled")
	} else {
		verb := fmt.Sprintf("am task add %s %s", req.Path, quote(req.Text))
		c := s.call(verb)
		if _, err := mcpserver.AddTask(context.Background(), s.root, mcpserver.AddTaskArgs{
			Path: req.Path, Text: req.Text,
		}); err != nil {
			http.Error(w, err.Error(), 400)
			c.fail(err)
			return
		}
		c.ok("task added")
	}
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- create-item ------------------------------------------------------------

type createItemReq struct {
	Backlog string `json:"backlog"`
	Title   string `json:"title"`
	Target  string `json:"target,omitempty"` // priority|icebox; defaults to icebox
}

func (s *server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req createItemReq
	if !decode(w, r, &req) {
		return
	}
	target := req.Target
	if target == "" {
		target = "icebox"
	}
	verb := fmt.Sprintf("am create-item %s --target %s", quote(req.Title), target)
	c := s.call(verb)
	res, err := mcpserver.CreateItem(context.Background(), s.root, mcpserver.CreateItemArgs{
		Backlog: req.Backlog, Title: req.Title,
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	// If the modal asked for priority placement, rank the freshly-
	// created story into priority. Default create-item lands in
	// icebox; the rank pull is what makes "create + start" feel
	// instant in the UI.
	if target == "priority" {
		_, _ = mcpserver.MoveToPriority(context.Background(), s.root, mcpserver.MoveToPriorityArgs{
			Backlog:   req.Backlog,
			ItemPaths: []string{res.Path},
			Position:  "top",
		})
	}
	c.ok("created " + res.Path)
	s.pokeSnapshot()
	writeOK(w, res)
}

// ---- delete-item ------------------------------------------------------------

type deleteItemReq struct {
	Path string `json:"path"`
}

func (s *server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	var req deleteItemReq
	if !decode(w, r, &req) {
		return
	}
	verb := "am delete-item " + req.Path
	c := s.call(verb)
	if err := callDeleteItem(s.root, req.Path); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("deleted " + req.Path)
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- block / unblock --------------------------------------------------------

func (s *server) handleBlock(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.BlockItemArgs
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am block %s --reason %s", req.Path, quote(req.Reason))
	c := s.call(verb)
	if _, err := mcpserver.BlockItem(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("blocked")
	s.pokeSnapshot()
	writeOK(w, nil)
}

func (s *server) handleUnblock(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.UnblockItemArgs
	if !decode(w, r, &req) {
		return
	}
	verb := "am unblock " + req.Path
	c := s.call(verb)
	if _, err := mcpserver.UnblockItem(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("unblocked")
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- reject ----------------------------------------------------------------

func (s *server) handleReject(w http.ResponseWriter, r *http.Request) {
	var req mcpserver.RejectItemArgs
	if !decode(w, r, &req) {
		return
	}
	verb := fmt.Sprintf("am reject %s --reason %s", req.Path, quote(req.Reason))
	c := s.call(verb)
	if _, err := mcpserver.RejectItem(context.Background(), s.root, req); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("rejected, back to started")
	s.pokeSnapshot()
	writeOK(w, nil)
}

// ---- sync ------------------------------------------------------------------

func (s *server) handleSync(w http.ResponseWriter, r *http.Request) {
	c := s.call("am sync")
	if _, err := mcpserver.Sync(context.Background(), s.root, mcpserver.SyncArgs{}); err != nil {
		http.Error(w, err.Error(), 400)
		c.fail(err)
		return
	}
	c.ok("sync complete")
	s.pokeSnapshot()
	writeOK(w, nil)
}
