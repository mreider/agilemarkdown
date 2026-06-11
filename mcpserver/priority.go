package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OrderRow struct {
	Index      int      `json:"index"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Status     string   `json:"status,omitempty"`
	Estimate   string   `json:"estimate,omitempty"`
	Type       string   `json:"type,omitempty"`
	Assignees  []string `json:"assignees,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Blocked    bool     `json:"blocked,omitempty"`
	CommentCnt int      `json:"comment_count,omitempty"`
	Epic       string   `json:"epic,omitempty"`
	// Transition timestamps. RFC3339 strings; empty when unset.
	// The client uses Accepted to group Done by historical iteration.
	Started   string `json:"started,omitempty"`
	Finished  string `json:"finished,omitempty"`
	Delivered string `json:"delivered,omitempty"`
	Accepted  string `json:"accepted,omitempty"`
	// ReleaseDate is populated for release-type stories only. It's the
	// target ship date the marker stakes in the timeline.
	ReleaseDate string `json:"release_date,omitempty"`
}

type PriorityListArgs struct {
	Backlog string `json:"backlog"`
}
type PriorityListResult struct {
	Backlog  string     `json:"backlog"`
	Items    []OrderRow `json:"items"`
	Count    int        `json:"count"`
	Velocity float64    `json:"velocity"`
}

func priorityListTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, PriorityListArgs) (*mcp.CallToolResult, PriorityListResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args PriorityListArgs) (*mcp.CallToolResult, PriorityListResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, PriorityListResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, PriorityListResult{}, err
		}
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return nil, PriorityListResult{}, err
		}
		byPath := indexItems(bck)

		cfg, _ := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		now := time.Now()

		out := make([]OrderRow, 0, pri.Len())
		for i, e := range pri.Entries() {
			it, ok := byPath[e.Path]
			if ok && cfg != nil && backlog.IsDone(it, cfg, now) {
				// Hidden from the active backlog view; surfaces via
				// done_list. Skipping keeps the iteration band math
				// honest.
				continue
			}
			row := OrderRow{Index: i, Title: e.Title, Path: e.Path}
			if ok {
				row.Title = it.Title()
				row.Status = it.Status()
				row.Estimate = it.Estimate()
				row.Type = it.Type()
				row.Assignees = it.Assignees()
				row.Tags = it.Tags()
				row.Blocked = it.Blocked()
				row.CommentCnt = len(it.Comments())
				row.Epic = it.Epic()
				if ts := it.Started(); !ts.IsZero() {
					row.Started = ts.Format(time.RFC3339)
				}
				if ts := it.Finished(); !ts.IsZero() {
					row.Finished = ts.Format(time.RFC3339)
				}
				if ts := it.Delivered(); !ts.IsZero() {
					row.Delivered = ts.Format(time.RFC3339)
				}
				if ts := it.Accepted(); !ts.IsZero() {
					row.Accepted = ts.Format(time.RFC3339)
				}
				if rd := it.ReleaseDate(); rd != "" {
					row.ReleaseDate = rd
				}
			}
			out = append(out, row)
		}

		velocity := computeVelocity(bck, cfg, root.Root())

		return nil, PriorityListResult{Backlog: args.Backlog, Items: out, Count: len(out), Velocity: velocity}, nil
	}
}

type DoneListArgs struct {
	Backlog string `json:"backlog"`
}
type DoneListResult struct {
	Backlog string     `json:"backlog"`
	Items   []OrderRow `json:"items"`
	Count   int        `json:"count"`
}

func doneListTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, DoneListArgs) (*mcp.CallToolResult, DoneListResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args DoneListArgs) (*mcp.CallToolResult, DoneListResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, DoneListResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, DoneListResult{}, err
		}
		cfg, _ := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if cfg == nil {
			return nil, DoneListResult{}, fmt.Errorf("config not loadable for %s", root.Root())
		}
		now := time.Now()

		type donePair struct {
			row  OrderRow
			when time.Time
		}
		var pairs []donePair
		for _, it := range bck.ActiveItems() {
			if !backlog.IsDone(it, cfg, now) {
				continue
			}
			row := OrderRow{
				Title: it.Title(), Path: filepath.Base(it.Path()),
				Status: it.Status(), Estimate: it.Estimate(),
				Type: it.Type(), Assignees: it.Assignees(),
				Tags: it.Tags(), Blocked: it.Blocked(),
				CommentCnt: len(it.Comments()), Epic: it.Epic(),
			}
			if ts := it.Started(); !ts.IsZero() {
				row.Started = ts.Format(time.RFC3339)
			}
			if ts := it.Finished(); !ts.IsZero() {
				row.Finished = ts.Format(time.RFC3339)
			}
			if ts := it.Delivered(); !ts.IsZero() {
				row.Delivered = ts.Format(time.RFC3339)
			}
			if ts := it.Accepted(); !ts.IsZero() {
				row.Accepted = ts.Format(time.RFC3339)
			}
			if rd := it.ReleaseDate(); rd != "" {
				row.ReleaseDate = rd
			}
			pairs = append(pairs, donePair{row: row, when: it.Accepted()})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].when.After(pairs[j].when) })
		out := make([]OrderRow, len(pairs))
		for i, p := range pairs {
			p.row.Index = i
			out[i] = p.row
		}
		return nil, DoneListResult{Backlog: args.Backlog, Items: out, Count: len(out)}, nil
	}
}

type IceboxListArgs struct {
	Backlog string `json:"backlog"`
}
type IceboxListResult struct {
	Backlog string     `json:"backlog"`
	Items   []OrderRow `json:"items"`
	Count   int        `json:"count"`
}

func iceboxListTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, IceboxListArgs) (*mcp.CallToolResult, IceboxListResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args IceboxListArgs) (*mcp.CallToolResult, IceboxListResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, IceboxListResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, IceboxListResult{}, err
		}
		ice, err := backlog.LoadIcebox(dir)
		if err != nil {
			return nil, IceboxListResult{}, err
		}
		byPath := indexItems(bck)
		out := make([]OrderRow, 0, ice.Len())
		for i, e := range ice.Entries() {
			row := OrderRow{Index: i, Title: e.Title, Path: e.Path}
			if it, ok := byPath[e.Path]; ok {
				row.Title = it.Title()
				row.Status = it.Status()
				row.Estimate = it.Estimate()
				row.Type = it.Type()
				row.Assignees = it.Assignees()
				row.Tags = it.Tags()
				row.Blocked = it.Blocked()
				row.CommentCnt = len(it.Comments())
				row.Epic = it.Epic()
			}
			out = append(out, row)
		}
		return nil, IceboxListResult{Backlog: args.Backlog, Items: out, Count: len(out)}, nil
	}
}

type RankItemArgs struct {
	Backlog  string `json:"backlog"`
	ItemPath string `json:"item_path" jsonschema:"item file basename inside the backlog"`
	Position string `json:"position,omitempty" jsonschema:"top|bottom"`
	After    string `json:"after,omitempty" jsonschema:"file basename to place this item after"`
	Before   string `json:"before,omitempty" jsonschema:"file basename to place this item before"`
}

func rankItemTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, RankItemArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args RankItemArgs) (*mcp.CallToolResult, OkResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, OkResult{}, err
		}
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return nil, OkResult{}, err
		}
		ice, err := backlog.LoadIcebox(dir)
		if err != nil {
			return nil, OkResult{}, err
		}
		item := basename(args.ItemPath)
		// pull from icebox if needed
		if pri.IndexOf(item) < 0 {
			if i := ice.IndexOf(item); i >= 0 {
				e := ice.Entries()[i]
				ice.Remove(item)
				pri.InsertBottom(e)
				if err := ice.Save(); err != nil {
					return nil, OkResult{}, err
				}
			} else {
				return nil, OkResult{}, fmt.Errorf("%s not in priority or icebox", item)
			}
		}
		switch {
		case strings.EqualFold(args.Position, "top"):
			pri.MoveTo(item, 0)
		case strings.EqualFold(args.Position, "bottom"):
			pri.MoveTo(item, pri.Len()-1)
		case args.After != "":
			pri.MoveAfter(item, basename(args.After))
		case args.Before != "":
			pri.MoveBefore(item, basename(args.Before))
		default:
			return nil, OkResult{}, fmt.Errorf("specify position (top|bottom) or after/before")
		}
		if err := pri.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type MoveToIceboxArgs struct {
	Backlog  string `json:"backlog"`
	ItemPath string `json:"item_path"`
	Position string `json:"position,omitempty" jsonschema:"top|bottom (default bottom)"`
}

func moveToIceboxTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, MoveToIceboxArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args MoveToIceboxArgs) (*mcp.CallToolResult, OkResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, OkResult{}, err
		}
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return nil, OkResult{}, err
		}
		ice, err := backlog.LoadIcebox(dir)
		if err != nil {
			return nil, OkResult{}, err
		}
		item := basename(args.ItemPath)
		idx := pri.IndexOf(item)
		if idx < 0 {
			return nil, OkResult{}, fmt.Errorf("%s not in priority", item)
		}
		// Integrity guard: a story the dev pair is mid-flight on does
		// not belong in the icebox. Started / finished / delivered all
		// imply work in progress; sending them back to the icebox
		// would make the board lie about reality.
		itemPath := filepath.Join(dir, item)
		if bi, err := backlog.LoadBacklogItem(itemPath); err == nil {
			switch bi.Status() {
			case backlog.StartedStatus.Name, backlog.FinishedStatus.Name, backlog.DeliveredStatus.Name:
				return nil, OkResult{}, fmt.Errorf("cannot move %s to icebox: status is %s. Unstart it first.", item, bi.Status())
			}
		}
		e := pri.Entries()[idx]
		pri.Remove(item)
		if strings.EqualFold(args.Position, "top") {
			ice.InsertTop(e)
		} else {
			ice.InsertBottom(e)
		}
		if err := pri.Save(); err != nil {
			return nil, OkResult{}, err
		}
		if err := ice.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type MoveToPriorityArgs struct {
	Backlog   string   `json:"backlog"`
	ItemPaths []string `json:"item_paths" jsonschema:"order is preserved when bulk-moving"`
	Position  string   `json:"position,omitempty" jsonschema:"top|bottom (default bottom). Multi-item moves always preserve input order."`
	After     string   `json:"after,omitempty" jsonschema:"single-item only: place after this item"`
}

func moveToPriorityTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, MoveToPriorityArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args MoveToPriorityArgs) (*mcp.CallToolResult, OkResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, OkResult{}, err
		}
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return nil, OkResult{}, err
		}
		ice, err := backlog.LoadIcebox(dir)
		if err != nil {
			return nil, OkResult{}, err
		}
		if len(args.ItemPaths) == 0 {
			return nil, OkResult{}, fmt.Errorf("item_paths is required")
		}
		// Pluck entries from icebox in input order.
		picked := make([]backlog.OrderEntry, 0, len(args.ItemPaths))
		for _, p := range args.ItemPaths {
			b := basename(p)
			idx := ice.IndexOf(b)
			if idx < 0 {
				return nil, OkResult{}, fmt.Errorf("%s not in icebox", b)
			}
			picked = append(picked, ice.Entries()[idx])
			ice.Remove(b)
		}
		// Single-item special case for `after`.
		if len(picked) == 1 && args.After != "" {
			anchor := basename(args.After)
			ai := pri.IndexOf(anchor)
			if ai < 0 {
				pri.InsertBottom(picked[0])
			} else {
				pri.InsertAt(ai+1, picked[0])
			}
		} else {
			top := strings.EqualFold(args.Position, "top")
			if top {
				// Insert in order at index 0..n.
				for i, e := range picked {
					pri.InsertAt(i, e)
				}
			} else {
				for _, e := range picked {
					pri.InsertBottom(e)
				}
			}
		}
		if err := pri.Save(); err != nil {
			return nil, OkResult{}, err
		}
		if err := ice.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type EpicProgressArgs struct {
	Slug string `json:"slug"`
}
type EpicProgressResult struct {
	Slug            string  `json:"slug"`
	TotalStories    int     `json:"total_stories"`
	AcceptedStories int     `json:"accepted_stories"`
	TotalPoints     float64 `json:"total_points"`
	AcceptedPoints  float64 `json:"accepted_points"`
	PercentDone     float64 `json:"percent_done"`
	ASCII           string  `json:"ascii"`
}

func epicProgressTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, EpicProgressArgs) (*mcp.CallToolResult, EpicProgressResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args EpicProgressArgs) (*mcp.CallToolResult, EpicProgressResult, error) {
		text, err := backlog.EpicASCII(root.Root(), args.Slug)
		if err != nil {
			return nil, EpicProgressResult{}, err
		}
		// Derive numeric counts by walking backlogs again (cheap; small repos).
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, EpicProgressResult{}, err
		}
		total, acc := 0, 0
		var totalPts, accPts float64
		for _, d := range dirs {
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return nil, EpicProgressResult{}, err
			}
			for _, it := range bck.ActiveItems() {
				if !strings.EqualFold(it.Epic(), args.Slug) {
					continue
				}
				total++
				pts := parsePoints(it.Estimate())
				totalPts += pts
				if strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
					acc++
					accPts += pts
				}
			}
		}
		pct := 0.0
		if totalPts > 0 {
			pct = accPts / totalPts * 100
		}
		return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, EpicProgressResult{
				Slug: args.Slug, TotalStories: total, AcceptedStories: acc,
				TotalPoints: totalPts, AcceptedPoints: accPts, PercentDone: pct, ASCII: text,
			}, nil
	}
}

type IterationViewArgs struct {
	Backlog string `json:"backlog"`
	Offset  int    `json:"offset,omitempty" jsonschema:"window offset; zero is the current iteration, one is next, and so on"`
}

func iterationViewTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, IterationViewArgs) (*mcp.CallToolResult, ChartResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args IterationViewArgs) (*mcp.CallToolResult, ChartResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, ChartResult{}, err
		}
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, ChartResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, ChartResult{}, err
		}
		text, err := backlog.IterationASCII(bck, dir, cfg, time.Now(), args.Offset)
		if err != nil {
			return nil, ChartResult{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, ChartResult{ASCII: text}, nil
	}
}

type RejectItemArgs struct {
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty" jsonschema:"optional reason for the rejection; recorded as a dated comment in the story body"`
	Author string `json:"author,omitempty" jsonschema:"PM nickname or name to attribute the rejection comment to; defaults to the item's author"`
}

// rejectItemTool implements the rejection event. Rejection is NOT a
// terminal status: it's a moment that gets logged. The story flips
// back to started so the dev pair can pick it up, fix what the PM
// flagged, and re-deliver. The reason lands as a dated comment so the
// history of the back-and-forth is preserved next to the story.
func rejectItemTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, RejectItemArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args RejectItemArgs) (*mcp.CallToolResult, OkResult, error) {
		item, itemRel, err := loadExistingItem(root, args.Path)
		if err != nil {
			return nil, OkResult{}, err
		}

		reason := strings.TrimSpace(args.Reason)
		text := "rejected, back to started"
		if reason != "" {
			text = "rejected, back to started. Reason: " + reason
		}
		author := strings.TrimSpace(args.Author)
		if author == "" {
			author = item.Author()
		}
		if author == "" {
			author = "PM"
		}
		// Record the event as a dated comment so the audit trail
		// lives next to the story under `## Comments`.
		item.SetBody(backlog.AppendComment(item.Body(), author, text))

		// Flip back to started so the dev pair can fix and resume.
		actions.ApplyStatusTransition(item, backlog.StartedStatus)

		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		recordActivity(root, author, "rejected", itemRel, reason)
		return nil, OkResult{OK: true}, nil
	}
}

func indexItems(bck *backlog.Backlog) map[string]*backlog.BacklogItem {
	out := make(map[string]*backlog.BacklogItem, len(bck.AllItems()))
	for _, it := range bck.ActiveItems() {
		out[filepath.Base(it.Path())] = it
	}
	return out
}

func basename(p string) string {
	p = strings.TrimSpace(p)
	if !strings.HasSuffix(p, ".md") {
		p += ".md"
	}
	return filepath.Base(p)
}

func computeVelocity(bck *backlog.Backlog, cfg *config.Config, rootDir string) float64 {
	if cfg == nil {
		return 0
	}
	var accepted []*backlog.BacklogItem
	for _, it := range bck.AllItems() {
		if backlog.CountsForVelocity(it, cfg) {
			accepted = append(accepted, it)
		}
	}
	overrides, _ := backlog.LoadIterationOverrides(rootDir)
	v, _, _ := backlog.ComputeVelocity(time.Now(), accepted, cfg, overrides)
	return v
}

func parsePoints(s string) float64 {
	// kept package-local; backlog.parsePoints is unexported.
	v := 0.0
	fmt.Sscanf(strings.TrimSpace(s), "%f", &v)
	return v
}
