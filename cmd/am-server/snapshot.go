package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/mcpserver"
)

// snapshotPayload mirrors what the VS Code extension sends to its
// webview on every snapshot push. The web bundle consumes the exact
// same shape via /ws/snapshot.
type snapshotPayload struct {
	Project    string                          `json:"project"`
	Backlog    string                          `json:"backlog"`
	Backlogs   []string                        `json:"backlogs"`
	Priority   mcpserver.PriorityListResult    `json:"priority"`
	Icebox     mcpserver.IceboxListResult      `json:"icebox"`
	Done       mcpserver.DoneListResult        `json:"done"`
	Dashboard  *mcpserver.DashboardResult      `json:"dashboard"`
	Velocity   mcpserver.VelocityHistoryResult `json:"velocity"`
	Epics      []mcpserver.EpicProgressResult  `json:"epics"`
}

func buildSnapshot(root string) (snapshotPayload, error) {
	ctx := context.Background()
	out := snapshotPayload{
		Project:  filepath.Base(root),
		Backlogs: []string{},
	}

	bl, err := mcpserver.ListBacklogs(ctx, root)
	if err != nil {
		return out, fmt.Errorf("list backlogs: %w", err)
	}
	out.Backlogs = bl.Backlogs
	if len(out.Backlogs) == 0 {
		return out, nil
	}
	// Single-backlog convention.
	first := out.Backlogs[0]
	out.Backlog = first

	out.Priority, err = mcpserver.PriorityList(ctx, root, mcpserver.PriorityListArgs{Backlog: first})
	if err != nil {
		return out, fmt.Errorf("priority: %w", err)
	}
	out.Icebox, err = mcpserver.IceboxList(ctx, root, mcpserver.IceboxListArgs{Backlog: first})
	if err != nil {
		return out, fmt.Errorf("icebox: %w", err)
	}
	out.Done, err = mcpserver.DoneList(ctx, root, mcpserver.DoneListArgs{Backlog: first})
	if err != nil {
		return out, fmt.Errorf("done: %w", err)
	}
	if d, err := mcpserver.Dashboard(ctx, root, mcpserver.DashboardArgs{}); err == nil {
		out.Dashboard = &d
	}
	if v, err := mcpserver.VelocityHistory(ctx, root, mcpserver.VelocityHistoryArgs{Backlog: first}); err == nil {
		out.Velocity = v
	}

	// Epic roll-ups across every distinct slug used by an item.
	slugs := map[string]struct{}{}
	for _, r := range out.Priority.Items {
		if r.Epic != "" {
			slugs[r.Epic] = struct{}{}
		}
	}
	for _, r := range out.Icebox.Items {
		if r.Epic != "" {
			slugs[r.Epic] = struct{}{}
		}
	}
	for _, r := range out.Done.Items {
		if r.Epic != "" {
			slugs[r.Epic] = struct{}{}
		}
	}
	for slug := range slugs {
		if ep, err := mcpserver.EpicProgress(ctx, root, mcpserver.EpicProgressArgs{Slug: slug}); err == nil {
			out.Epics = append(out.Epics, ep)
		}
	}

	// Silence the unused-import warning from the dot above (and
	// reserves the package for upcoming phases that read directly
	// from backlog.* helpers).
	_ = backlog.UnstartedStatus

	return out, nil
}
