package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
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

		out := make([]OrderRow, 0, pri.Len())
		for i, e := range pri.Entries() {
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

		cfg, _ := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		velocity := computeVelocity(bck, cfg, root.Root())

		return nil, PriorityListResult{Backlog: args.Backlog, Items: out, Count: len(out), Velocity: velocity}, nil
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
	Slug          string `json:"slug"`
	TotalStories  int    `json:"total_stories"`
	AcceptedStories int  `json:"accepted_stories"`
	TotalPoints   float64 `json:"total_points"`
	AcceptedPoints float64 `json:"accepted_points"`
	PercentDone   float64 `json:"percent_done"`
	ASCII         string  `json:"ascii"`
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
	Path          string `json:"path"`
	Reason        string `json:"reason,omitempty" jsonschema:"optional reason; appended to the item body under '## Rejection notes'"`
	FailingBullet int    `json:"failing_bullet,omitempty" jsonschema:"1-based index of the acceptance bullet that failed; cited in the rejection note and reopened from claimed to open"`
}

func rejectItemTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, RejectItemArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args RejectItemArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		actions.ApplyStatusTransition(item, backlog.RejectedStatus)

		// Resolve the failing bullet (if any) before the rejection note is
		// appended so the citation includes the bullet's text.
		var failingText string
		if args.FailingBullet > 0 {
			bullets := backlog.ParseAcceptance(item.Body())
			for _, b := range bullets {
				if b.Index == args.FailingBullet {
					failingText = b.Text
					break
				}
			}
			if failingText == "" {
				return nil, OkResult{}, fmt.Errorf("failing_bullet %d not found in body", args.FailingBullet)
			}
			// Flip the cited bullet back from claimed to open so the
			// ledger shows the contested bullet as still owing work.
			if body, err := backlog.SetAcceptanceState(item.Body(), args.FailingBullet, backlog.AcceptanceOpen, ""); err == nil {
				item.SetBody(body)
			}
		}

		reason := strings.TrimSpace(args.Reason)
		if reason != "" || failingText != "" {
			body := item.Body()
			now := time.Now().UTC().Format("2006-01-02")
			var line string
			switch {
			case failingText != "" && reason != "":
				line = fmt.Sprintf("- %s: Acceptance bullet %d (%q) failed. Reason: %s", now, args.FailingBullet, failingText, reason)
			case failingText != "":
				line = fmt.Sprintf("- %s: Acceptance bullet %d (%q) failed.", now, args.FailingBullet, failingText)
			default:
				line = fmt.Sprintf("- %s: %s", now, reason)
			}
			block := "\n\n## Rejection notes\n\n" + line + "\n"
			if !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			item.SetBody(body + block)
		}
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
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
