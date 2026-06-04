package mcpserver

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
)

// recordActivity appends a line to the audit trail (`.am/activity.log`)
// for a state change driven through the MCP server, mirroring the CLI's
// recordActivity so both surfaces feed one chronological feed. relPath
// is the item path relative to the repo root (the form MCP args already
// use); pass "" for repo-wide events. Best-effort: errors are ignored so
// the trail can never fail the tool call.
func recordActivity(root *backlog.BacklogsStructure, actor, verb, relPath, detail string) {
	if actor == "" {
		if name, _, err := git.CurrentUser(); err == nil && name != "" {
			actor = name
		} else {
			actor = "unknown"
		}
	}
	_ = backlog.AppendActivity(root.Root(), actor, verb, relPath, detail)
}
