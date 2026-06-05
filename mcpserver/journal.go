package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// JournalArgs optionally narrows the journal to stories whose path
// contains a substring.
type JournalArgs struct {
	Item string `json:"item,omitempty" jsonschema:"only stories whose path contains this substring"`
}

type JournalEvent struct {
	Time   string `json:"time"`
	Actor  string `json:"actor"`
	Verb   string `json:"verb"`
	Detail string `json:"detail,omitempty"`
}

type JournalStory struct {
	Item   string         `json:"item"`
	Title  string         `json:"title"`
	Events []JournalEvent `json:"events"`
}

// JournalResult is the narrative audit trail grouped by story, plus the
// active workflow mode. It is the autonomous-mode review surface: a human
// reads it after an unattended run to reconstruct what the agent did and
// why, rather than gating each story in real time.
type JournalResult struct {
	Mode    string         `json:"mode"`
	Stories []JournalStory `json:"stories"`
}

func journalTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, JournalArgs) (*mcp.CallToolResult, JournalResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args JournalArgs) (*mcp.CallToolResult, JournalResult, error) {
		entries, err := backlog.LoadActivity(root.Root())
		if err != nil {
			return nil, JournalResult{}, err
		}
		if filter := strings.TrimSpace(args.Item); filter != "" {
			kept := entries[:0:0]
			for _, e := range entries {
				if strings.Contains(e.Item, filter) {
					kept = append(kept, e)
				}
			}
			entries = kept
		}
		res := JournalResult{Mode: config.ModeCoached}
		if cfg, _ := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml")); cfg != nil {
			res.Mode = cfg.EffectiveMode()
		}
		for _, g := range backlog.GroupActivityByItem(entries) {
			s := JournalStory{Item: g.Item, Title: journalTitle(root.Root(), g.Item)}
			for _, e := range g.Entries {
				s.Events = append(s.Events, JournalEvent{
					Time:   e.Time.Format("2006-01-02T15:04:05Z07:00"),
					Actor:  e.Actor,
					Verb:   e.Verb,
					Detail: e.Detail,
				})
			}
			res.Stories = append(res.Stories, s)
		}
		return nil, res, nil
	}
}

// journalTitle resolves an item's title from disk, falling back to the
// path's base name (the story may have been archived or removed).
func journalTitle(root, item string) string {
	if item == "" {
		return "(project)"
	}
	if it, err := backlog.LoadBacklogItem(filepath.Join(root, item)); err == nil {
		if t := strings.TrimSpace(it.Title()); t != "" {
			return t
		}
	}
	return filepath.Base(item)
}

// SetModeArgs sets the project workflow mode.
type SetModeArgs struct {
	Mode string `json:"mode" jsonschema:"coached (dev pair stages, PM accepts) | autonomous (dev pair may accept its own work)"`
}

func setModeTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetModeArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetModeArgs) (*mcp.CallToolResult, OkResult, error) {
		want := strings.ToLower(strings.TrimSpace(args.Mode))
		if want != config.ModeCoached && want != config.ModeAutonomous {
			return nil, OkResult{}, fmt.Errorf("mode must be %s or %s, got %q", config.ModeCoached, config.ModeAutonomous, args.Mode)
		}
		cfgPath := filepath.Join(root.Root(), ".am", "config.yaml")
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return nil, OkResult{}, err
		}
		cfg.Mode = want
		if err := cfg.Save(cfgPath); err != nil {
			return nil, OkResult{}, err
		}
		recordActivity(root, "", "mode", "", want)
		return nil, OkResult{OK: true}, nil
	}
}
