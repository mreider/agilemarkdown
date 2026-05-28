package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/mcpserver"
)

// callBox carries the per-request server context every handler needs.
// Verb is the rendered CLI line that lands in the CommandLog feed.
type callBox struct {
	server *server
	verb   string
	out    string
}

func (s *server) call(verb string) *callBox {
	return &callBox{server: s, verb: verb}
}

func (c *callBox) ok(summary string) {
	c.out = summary
	c.server.log.emit(c.verb, true, summary)
}

func (c *callBox) fail(err error) error {
	c.server.log.emit(c.verb, false, err.Error())
	return err
}

// ---- set-release-date: handled in-process because no MCP wrapper.

func callSetReleaseDate(root, itemPath, date string) error {
	abs := filepath.Join(root, itemPath)
	item, err := backlog.LoadBacklogItem(abs)
	if err != nil {
		return fmt.Errorf("load item: %w", err)
	}
	item.SetReleaseDate(date)
	if err := item.Save(); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return nil
}

// ---- set-status now goes through the MCP wrapper so autoRank fires.

func callSetStatus(root, itemPath, status string) error {
	ctx := context.Background()
	_, err := mcpserver.SetStatus(ctx, root, mcpserver.SetStatusArgs{
		Path: itemPath, Status: status,
	})
	return err
}

// ---- set-type: no MCP wrapper; load + SetType + save.

func callSetType(root, itemPath, typ string) error {
	t := strings.ToLower(strings.TrimSpace(typ))
	switch t {
	case "feature", "bug", "chore", "release":
	default:
		return fmt.Errorf("invalid type %q (feature|bug|chore|release)", typ)
	}
	abs := filepath.Join(root, itemPath)
	item, err := backlog.LoadBacklogItem(abs)
	if err != nil {
		return fmt.Errorf("load item: %w", err)
	}
	item.SetType(t)
	return item.Save()
}

// ---- delete-item: also CLI-only; remove the file and prune any
// order-file entry that points at it.

func callDeleteItem(root, itemPath string) error {
	abs := filepath.Join(root, itemPath)
	// Sanity-guard: must live under root.
	if !strings.HasPrefix(abs, filepath.Clean(root)) {
		return fmt.Errorf("refusing to delete outside root")
	}
	dir := filepath.Dir(abs)
	base := filepath.Base(abs)
	// Remove the story file.
	if err := removeFile(abs); err != nil {
		return err
	}
	// Prune from _priority.md and _icebox.md if present.
	if pri, err := backlog.LoadPriority(dir); err == nil {
		if pri.IndexOf(base) >= 0 {
			pri.Remove(base)
			_ = pri.Save()
		}
	}
	if ice, err := backlog.LoadIcebox(dir); err == nil {
		if ice.IndexOf(base) >= 0 {
			ice.Remove(base)
			_ = ice.Save()
		}
	}
	return nil
}

func removeFile(path string) error {
	return os.Remove(path)
}

// ---- summary helpers ----

func summarizeItem(root, itemPath string) string {
	abs := filepath.Join(root, itemPath)
	item, err := backlog.LoadBacklogItem(abs)
	if err != nil {
		return ""
	}
	parts := []string{"status: " + item.Status()}
	if e := strings.TrimSpace(item.Estimate()); e != "" {
		parts = append(parts, "estimate: "+e)
	}
	return strings.Join(parts, "   ")
}
