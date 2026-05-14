package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stateMu serializes every state-mutating MCP tool call. The MCP SDK
// dispatches requests concurrently; without a mutex two tools racing on
// the same item file can clobber each other (e.g. change_tag and
// delete_tag back-to-back). Wrapped via locked() in handler files.
var stateMu sync.Mutex

// locked wraps a handler so it acquires stateMu for the duration of the
// call. Use on every handler that writes to disk.
func locked[A, R any](h func(context.Context, *mcp.CallToolRequest, A) (*mcp.CallToolResult, R, error)) func(context.Context, *mcp.CallToolRequest, A) (*mcp.CallToolResult, R, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args A) (*mcp.CallToolResult, R, error) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return h(ctx, req, args)
	}
}

// Run starts an MCP stdio server rooted at the given backlog root directory.
// rootDir must contain backlog folders (or be the parent git repo).
func Run(ctx context.Context, rootDir, version string) error {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return err
	}
	srv := buildServer(abs, version)
	return srv.Run(ctx, &mcp.StdioTransport{})
}

// buildServer wires up the MCP server with all tools but does not bind a
// transport. Exposed for tests that swap in an in-memory pair.
func buildServer(rootDir, version string) *mcp.Server {
	root := backlog.NewBacklogsStructure(rootDir)

	srv := mcp.NewServer(&mcp.Implementation{Name: "agilemarkdown", Version: version}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_backlogs",
		Description: "List all backlog folders in the project root.",
	}, listBacklogs(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_items",
		Description: "List items in a backlog. Optionally filter by status (unstarted|started|finished|delivered|accepted|rejected) or tag.",
	}, listItems(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_item",
		Description: "Read the full markdown body of a single backlog item by file path (relative to project root).",
	}, getItem(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_item",
		Description: "Create a new backlog item under a given backlog with a title.",
	}, locked(createItem(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_status",
		Description: "Change the status of an item identified by its file path. Status must be one of: unstarted, started, finished, delivered, accepted, rejected. Auto-stamps `finished`/`delivered`/`accepted` timestamps on transitions; clears them on regression.",
	}, locked(setStatus(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_assigned",
		Description: "Assign an item to a user (name or email).",
	}, locked(setAssigned(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_estimate",
		Description: "Set the story-point estimate on an item.",
	}, locked(setEstimate(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "validate",
		Description: "Run schema validation across all items. Returns the list of validation errors.",
	}, validateAll(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "sync",
		Description: "Run the full sync: regenerate index, velocity, timeline, tag pages; commit; push if a remote is configured.",
	}, locked(syncAction(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "velocity_chart",
		Description: "Render an ASCII bar chart of velocity (accepted points per iteration) for a backlog. Renders inline in any MCP client (Claude Desktop, Code, Cursor).",
	}, velocityChart(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "timeline_chart",
		Description: "Render an ASCII Gantt for items carrying a given tag. Items must have a `timeline.start` and `timeline.end`.",
	}, timelineChart(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_backlog",
		Description: "Create a new backlog folder under the project root with sample feature/bug/chore items. The backlog name becomes a top-level directory plus an overview file (`<name>.md`).",
	}, locked(createBacklogTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "archive_items",
		Description: "Archive every active item whose Modified date falls on or before `before` (YYYY-MM-DD). Items are flagged with `archive: true` and move to <backlog>/archive/ on next sync. Use to keep the active backlog small after a release.",
	}, locked(archiveItemsTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "team_agreements",
		Description: "Read (or with `set`, overwrite) team-agreements.md at the project root. Single source of truth for working agreements; the place to record rules an LLM agent should follow alongside the team.",
	}, locked(teamAgreementsTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "record_learning",
		Description: "Append a one-line learning entry to learnings.md at the project root. Date is prefixed automatically. Use for removal-experiment outcomes, scratch-refactor takeaways, or retro one-liners.",
	}, locked(recordLearningTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_hypothesis",
		Description: "Deprecated. Set the `hypothesis:` frontmatter on an item. The Pivotal way uses acceptance criteria in the body's `## Acceptance` section, not a hypothesis field; the coach reads those criteria. This tool is kept for back-compat.",
	}, locked(setHypothesisTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_tags",
		Description: "Replace the `tags:` list on an item. Pass an empty array to clear. Use `change_tag`/`delete_tag` for fleet-wide rename/drop.",
	}, locked(setTagsTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_epic",
		Description: "Set the `epic:` slug on an item. Pass an empty slug to clear. Stories sharing a slug roll up under `epic_progress`.",
	}, locked(setEpicTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "change_tag",
		Description: "Rename a tag fleet-wide: every item carrying `old` is rewritten to use `new`. Run `sync` afterwards to regenerate tag pages.",
	}, locked(changeTagTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_tag",
		Description: "Remove a tag from every item that carries it. Returns the number of items modified. Run `sync` afterwards to regenerate tag pages.",
	}, locked(deleteTagTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_iteration_override",
		Description: "Set per-iteration team-strength and/or length overrides in `.am/iterations.yaml`. Mirrors Pivotal Tracker's iteration_override resource. team_strength is a float (1.0 = full strength, 0 = excluded from velocity, >1.0 allowed). Pass `unset: true` to remove the record. Strength normalizes the velocity formula: SUM(points/strength)/SUM(length).",
	}, locked(setIterationOverrideTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_iteration_overrides",
		Description: "Return every active iteration override (number, team_strength, length_weeks). Empty list if `.am/iterations.yaml` is absent.",
	}, listIterationOverridesTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "cycle_time_chart",
		Description: "Render a cycle-time summary for a backlog: median time from started to accepted plus the five longest stories. Releases excluded.",
	}, cycleTimeChartTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "rejection_rate",
		Description: "Per-iteration rejection rate over the rolling lookback window. Returns rows with accepted, rejected, percent. Pivotal target band is 5-15%.",
	}, rejectionRateTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "priority_list",
		Description: "Return the ordered contents of a backlog's _priority.md (top = highest rank). Includes status, points, type per row plus the project velocity so a client can draw iteration bands.",
	}, priorityListTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "icebox_list",
		Description: "Return the ordered contents of a backlog's _icebox.md (top = highest rank).",
	}, iceboxListTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "rank_item",
		Description: "Reorder an item inside _priority.md. Provide `position` (top|bottom) or `after`/`before` (a sibling file basename). If the item is currently in icebox, it is pulled into priority first.",
	}, locked(rankItemTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "move_to_icebox",
		Description: "Move an item from _priority.md to _icebox.md. `position` is top or bottom (default bottom).",
	}, locked(moveToIceboxTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "move_to_priority",
		Description: "Bulk-move items from _icebox.md to _priority.md. Input order is preserved. With a single item, `after` lets you place it precisely; otherwise `position` (top|bottom) controls placement.",
	}, locked(moveToPriorityTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "epic_progress",
		Description: "Render an ASCII burnup for an epic identified by slug. Walks every backlog and rolls up stories whose `epic:` frontmatter equals the slug.",
	}, epicProgressTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "iteration_view",
		Description: "Render a single iteration window from _priority.md. Offset 0 is the current iteration; 1 is next; etc. Velocity-bounded.",
	}, iterationViewTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "reject_item",
		Description: "Reject an item: transitions status to `rejected` and (with `reason`) appends a dated note under '## Rejection notes' in the item body. Use this instead of set_status when capturing PM rejection rationale.",
	}, locked(rejectItemTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "block_item",
		Description: "Mark an item as blocked. Stores `blocked: true` plus an optional `blocked_reason:` in frontmatter. The item still flows through the state machine; UIs render a blocked badge.",
	}, locked(blockItemTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "unblock_item",
		Description: "Clear an item's blocked flag and reason.",
	}, locked(unblockItemTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_comments",
		Description: "Return parsed comments for an item (author, when, text). Comments live under '## Comments' in the body; this tool returns one row per comment plus a count for badge rendering.",
	}, getCommentsTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "add_comment",
		Description: "Append a comment under '## Comments' in an item's body. Author defaults to the item's `author:` frontmatter or the literal 'user'. Date is stamped automatically.",
	}, locked(addCommentTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_tasks",
		Description: "Return parsed checkbox tasks under '## Tasks' in an item body. Index is 1-based and stable per parse.",
	}, listTasksTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "add_task",
		Description: "Append an unchecked task under '## Tasks' in an item body. Creates the section if missing.",
	}, locked(addTaskTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_task_done",
		Description: "Flip the checkbox of the task at 1-based index to `done` (true|false).",
	}, locked(setTaskDoneTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "burnup_chart",
		Description: "Per-day burnup rows for one iteration window. Returns rows of {day, scope, done} plus the iteration start and end. `offset` selects the window: 0 is current, 1 is next, -1 is previous.",
	}, burnupChartTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "type_mix",
		Description: "Counts and percentages of accepted stories by type (feature, bug, chore, release) over the rolling lookback window.",
	}, typeMixTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "cumulative_flow",
		Description: "Per-day cumulative-flow rows: count of accepted stories vs open (not-yet-accepted) stories over the last N days (default 30). Project-level.",
	}, cfdTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "search",
		Description: "Substring search across all active stories. Scores title, tags, path, and body matches and returns the top hits with short snippets.",
	}, searchTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "velocity_history",
		Description: "Structured velocity history (one row per completed iteration in the lookback). Each row has iteration number, start date, planned points, accepted points, length_weeks and team_strength.",
	}, velocityHistoryTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "next_item",
		Description: "Highest-ranked unstarted, unblocked story across the project (or one backlog when filtered). The 'next pull' answer.",
	}, nextItemTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "dashboard",
		Description: "One-block project dashboard: latest velocity, volatility percent, median cycle time in hours, latest rejection-rate percent, total stories accepted.",
	}, dashboardTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_description",
		Description: "Replace the markdown body of an item. The frontmatter block is preserved. Use for full edits from a UI; comments and tasks live inside the body, so callers must include them.",
	}, locked(setDescriptionTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "coach_check",
		Description: "Preflight a planned action against Pivotal canon. Returns a structured verdict: allowed/refused, the rule, the canonical essay slug, and a suggested next move. Use before set_status to accepted, set_estimate, or create_item with an estimate.",
	}, coachCheckTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "acceptance_prompt",
		Description: "Render the PM acceptance ceremony for one delivered story. Returns title, type, hypothesis, estimate, and the verification bullets pulled from the body's Acceptance section. The agent shows this to the human and waits for an answer; the answer drives set_status or reject_item.",
	}, acceptancePromptTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_acceptance",
		Description: "Return the parsed acceptance bullets for one story. Each bullet has a 1-based index, a state (open, claimed, verified), the bullet text, and an optional claim note left by the dev pair. Use before set_acceptance_state so the index references the body as it stands now.",
	}, listAcceptanceTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_acceptance_state",
		Description: "Flip one acceptance bullet's state. The agent marks bullets claimed at delivery time (optionally with a claim note); the PM ceremony marks them verified at acceptance time. Indices are 1-based and only valid against the body as it was when list_acceptance was called.",
	}, setAcceptanceStateTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "append_acceptance_bullet",
		Description: "Append a new open acceptance bullet to a story. Creates the Acceptance section if one does not exist. Used by skills that draft criteria (am-decompose, am-plan) into an existing body.",
	}, appendAcceptanceBulletTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "iteration_fit",
		Description: "Report whether the current iteration's planned points fit within the rolling velocity. Optionally adds a candidate item to the planned total to forecast the impact of pulling in another story.",
	}, iterationFitTool(root))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "inception_doc",
		Description: "Read or write the project's inception.md (one-page narrative covering the user, the goal, the reason, success, constraints, out of scope). Empty body = read; non-empty = write. Returns existed/wrote flags.",
	}, locked(inceptionDocTool(root)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "sprint_plan",
		Description: "Render the iteration plan: top of priority up to rolling velocity, plus a below-line backlog. Flags stories missing `## Acceptance`, oversized features, unestimated features, and overcommit. The PM uses this at IPM to confirm the rank order before starting the iteration.",
	}, sprintPlanTool(root))

	return srv
}

// --- tool input/output types ---

type ListBacklogsArgs struct{}

type ListBacklogsResult struct {
	Backlogs []string `json:"backlogs"`
}

type ListItemsArgs struct {
	Backlog string `json:"backlog" jsonschema:"name of the backlog folder"`
	Status  string `json:"status,omitempty" jsonschema:"optional status filter: unstarted, started, finished, delivered, accepted, or rejected"`
	Tag     string `json:"tag,omitempty" jsonschema:"optional tag filter"`
}

type ItemSummary struct {
	Path       string   `json:"path"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Type       string   `json:"type,omitempty"`
	Assigned   string   `json:"assigned,omitempty"`
	Assignees  []string `json:"assignees,omitempty"`
	Estimate   string   `json:"estimate,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Blocked    bool     `json:"blocked,omitempty"`
	CommentCnt int      `json:"comment_count,omitempty"`
	Epic       string   `json:"epic,omitempty"`
}

type ListItemsResult struct {
	Items []ItemSummary `json:"items"`
	Count int           `json:"count"`
}

type GetItemArgs struct {
	Path string `json:"path" jsonschema:"file path relative to project root"`
}

type GetItemResult struct {
	Path           string                `json:"path"`
	Title          string                `json:"title"`
	Status         string                `json:"status"`
	Type           string                `json:"type,omitempty"`
	Assigned       string                `json:"assigned,omitempty"`
	Assignees      []string              `json:"assignees,omitempty"`
	Estimate       string                `json:"estimate,omitempty"`
	Tags           []string              `json:"tags,omitempty"`
	Blocked        bool                  `json:"blocked,omitempty"`
	BlockedReason  string                `json:"blocked_reason,omitempty"`
	Epic           string                `json:"epic,omitempty"`
	Author         string                `json:"author,omitempty" jsonschema:"original reporter from item frontmatter"`
	Started        string                `json:"started,omitempty" jsonschema:"YYYY-MM-DD when status first hit started"`
	Finished       string                `json:"finished,omitempty"`
	Delivered      string                `json:"delivered,omitempty"`
	Accepted       string                `json:"accepted,omitempty"`
	Iteration      int                   `json:"iteration,omitempty" jsonschema:"iteration number; 0 when unknown"`
	IterationLabel string                `json:"iteration_label,omitempty" jsonschema:"short label for items without a numeric iteration: backlog, icebox, in flight"`
	Body           string                `json:"body"`
	Acceptance     []AcceptanceBulletRow `json:"acceptance,omitempty" jsonschema:"parsed acceptance bullets if the body has an Acceptance section"`
}

type CreateItemArgs struct {
	Backlog string `json:"backlog" jsonschema:"backlog folder name"`
	Title   string `json:"title" jsonschema:"item title"`
	User    string `json:"user,omitempty" jsonschema:"author user name or email (optional)"`
}

type CreateItemResult struct {
	Path string `json:"path"`
}

type SetStatusArgs struct {
	Path   string `json:"path"`
	Status string `json:"status" jsonschema:"one of: unstarted, started, finished, delivered, accepted, rejected"`
}

type SetAssignedArgs struct {
	Path      string   `json:"path"`
	Assigned  string   `json:"assigned,omitempty" jsonschema:"single assignee (back-compat); ignored when assignees is set"`
	Assignees []string `json:"assignees,omitempty" jsonschema:"list of up to 3 assignees"`
}

type SetEstimateArgs struct {
	Path     string `json:"path"`
	Estimate string `json:"estimate"`
}

type OkResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type ValidateResult struct {
	Errors []string `json:"errors"`
}

type SyncArgs struct{}

// --- tool handlers ---

func listBacklogs(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ListBacklogsArgs) (*mcp.CallToolResult, ListBacklogsResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ListBacklogsArgs) (*mcp.CallToolResult, ListBacklogsResult, error) {
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, ListBacklogsResult{}, err
		}
		// `BacklogDirs()` returns every non-hidden subdirectory because
		// `sync` needs candidates that don't yet have an overview file.
		// For machine-readable consumers (the VS Code extension, scripts)
		// only return directories that are validated backlogs: the ones
		// with `<name>.md` overview in the project root. Drops `src/`,
		// `docs/`, etc. from projects where AM lives alongside code.
		names := make([]string, 0, len(dirs))
		for _, d := range dirs {
			if _, ok := backlog.FindOverviewFileInRootDirectory(d); !ok {
				continue
			}
			names = append(names, filepath.Base(d))
		}
		return nil, ListBacklogsResult{Backlogs: names}, nil
	}
}

func listItems(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ListItemsArgs) (*mcp.CallToolResult, ListItemsResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ListItemsArgs) (*mcp.CallToolResult, ListItemsResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, ListItemsResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, ListItemsResult{}, err
		}
		items := bck.ActiveItems()
		out := make([]ItemSummary, 0, len(items))
		for _, item := range items {
			if args.Status != "" && !strings.EqualFold(item.Status(), args.Status) {
				continue
			}
			tags := item.Tags()
			if args.Tag != "" {
				match := false
				for _, t := range tags {
					if strings.EqualFold(t, args.Tag) {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
			rel, _ := filepath.Rel(root.Root(), item.Path())
			assignees := item.Assignees()
			out = append(out, ItemSummary{
				Path:       rel,
				Title:      item.Title(),
				Status:     item.Status(),
				Type:       item.Type(),
				Assigned:   item.Assigned(),
				Assignees:  assignees,
				Estimate:   item.Estimate(),
				Tags:       tags,
				Blocked:    item.Blocked(),
				CommentCnt: len(item.Comments()),
				Epic:       item.Epic(),
			})
		}
		return nil, ListItemsResult{Items: out, Count: len(out)}, nil
	}
}

func getItem(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, GetItemArgs) (*mcp.CallToolResult, GetItemResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args GetItemArgs) (*mcp.CallToolResult, GetItemResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, GetItemResult{}, err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, GetItemResult{}, err
		}
		// Iteration is derived server-side from item timestamps when
		// possible. Unstarted items in priority get a 0 here; the
		// extension fills that in from priority position + velocity.
		iteration := 0
		iterationLabel := ""
		if cfg, cerr := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml")); cerr == nil {
			iteration, iterationLabel = backlog.ItemIteration(item, cfg, 0, 0)
		}
		return nil, GetItemResult{
			Path:           args.Path,
			Title:          item.Title(),
			Status:         item.Status(),
			Type:           item.Type(),
			Assigned:       item.Assigned(),
			Assignees:      item.Assignees(),
			Estimate:       item.Estimate(),
			Tags:           item.Tags(),
			Blocked:        item.Blocked(),
			BlockedReason:  item.BlockedReason(),
			Epic:           item.Epic(),
			Author:         item.Author(),
			Started:        formatTimestamp(item.Started()),
			Finished:       formatTimestamp(item.Finished()),
			Delivered:      formatTimestamp(item.Delivered()),
			Accepted:       formatTimestamp(item.Accepted()),
			Iteration:      iteration,
			IterationLabel: iterationLabel,
			Body:           string(body),
			Acceptance:     bulletsToRows(backlog.ParseAcceptance(item.Body())),
		}, nil
	}
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func createItem(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, CreateItemArgs) (*mcp.CallToolResult, CreateItemResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args CreateItemArgs) (*mcp.CallToolResult, CreateItemResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, CreateItemResult{}, err
		}
		action := actions.NewCreateItemAction(dir, args.Title, args.User, false)
		if err := action.Execute(); err != nil {
			return nil, CreateItemResult{}, err
		}
		fileName := utils.GetValidFileName(args.Title) + ".md"
		rel, _ := filepath.Rel(root.Root(), filepath.Join(dir, fileName))

		// Stage the new item into _icebox.md so it shows up in the
		// board (and any other priority/icebox view) without requiring
		// the caller to run sync first. `am sync` would do the same
		// auto-icebox step for orphan items, but the caller of
		// `create_item` has no reason to know that. Keep the new item
		// out of _priority.md; the icebox is the single ingress
		// funnel.
		if ice, err := backlog.LoadIcebox(dir); err == nil {
			if ice.IndexOf(fileName) < 0 {
				if pri, perr := backlog.LoadPriority(dir); perr != nil || pri.IndexOf(fileName) < 0 {
					ice.InsertBottom(backlog.OrderEntry{Title: args.Title, Path: fileName})
					_ = ice.Save()
				}
			}
		}

		return nil, CreateItemResult{Path: rel}, nil
	}
}

func setStatus(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetStatusArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetStatusArgs) (*mcp.CallToolResult, OkResult, error) {
		st := backlog.StatusByName(args.Status)
		if st == nil {
			return nil, OkResult{}, fmt.Errorf("invalid status %q (valid: %s)", args.Status, backlog.AllStatusesList())
		}
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		actions.ApplyStatusTransition(item, st)
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

func setAssigned(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetAssignedArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetAssignedArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		var xs []string
		if len(args.Assignees) > 0 {
			xs = args.Assignees
		} else if args.Assigned != "" {
			xs = []string{args.Assigned}
		}
		if len(xs) > 3 {
			return nil, OkResult{}, fmt.Errorf("at most 3 assignees allowed; got %d", len(xs))
		}
		item.SetAssignees(xs)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

func setEstimate(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetEstimateArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetEstimateArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetEstimate(args.Estimate)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

func validateAll(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, ValidateResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ValidateResult, error) {
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, ValidateResult{}, err
		}
		var msgs []string
		for _, dir := range dirs {
			bck, err := backlog.LoadBacklog(dir)
			if err != nil {
				return nil, ValidateResult{}, err
			}
			for _, item := range bck.AllItems() {
				for _, e := range backlog.ValidateItem(item) {
					rel, _ := filepath.Rel(root.Root(), e.Path)
					msgs = append(msgs, fmt.Sprintf("%s: %s: %s", rel, e.Key, e.Message))
				}
			}
		}
		return nil, ValidateResult{Errors: msgs}, nil
	}
}

func syncAction(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SyncArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ SyncArgs) (*mcp.CallToolResult, OkResult, error) {
		err := actions.NewSyncAction(root.Root(), "", false).Execute()
		if err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true, Message: "sync complete"}, nil
	}
}

// --- v2 tool handlers (charts) ---

type VelocityChartArgs struct {
	Backlog        string `json:"backlog"`
	IterationCount int    `json:"iteration_count,omitempty" jsonschema:"defaults to 12"`
}
type ChartResult struct {
	ASCII string `json:"ascii"`
}

func velocityChart(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, VelocityChartArgs) (*mcp.CallToolResult, ChartResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args VelocityChartArgs) (*mcp.CallToolResult, ChartResult, error) {
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
		count := args.IterationCount
		if count <= 0 {
			count = 12
		}
		overrides, _ := backlog.LoadIterationOverrides(root.Root())
		text := backlog.VelocityASCII(bck, count, cfg, overrides)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, ChartResult{ASCII: text}, nil
	}
}

type TimelineChartArgs struct {
	Tag string `json:"tag"`
}

func timelineChart(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, TimelineChartArgs) (*mcp.CallToolResult, ChartResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args TimelineChartArgs) (*mcp.CallToolResult, ChartResult, error) {
		_, itemsTags, _, err := backlog.ItemsTags(root)
		if err != nil {
			return nil, ChartResult{}, err
		}
		text := backlog.TimelineASCII(itemsTags[args.Tag], args.Tag)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, ChartResult{ASCII: text}, nil
	}
}

func resolveBacklogDir(root *backlog.BacklogsStructure, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("backlog name required")
	}
	dirs, err := root.BacklogDirs()
	if err != nil {
		return "", err
	}
	for _, d := range dirs {
		if filepath.Base(d) == name {
			return d, nil
		}
	}
	return "", fmt.Errorf("backlog %q not found", name)
}
