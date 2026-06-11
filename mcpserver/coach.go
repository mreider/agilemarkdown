package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CoachCheckArgs is the preflight a planning agent runs before a state
// transition or write. Mirrors the canon described in the
// `coach-refuses-pm-accepts` essay.
type CoachCheckArgs struct {
	Action   string `json:"action" jsonschema:"set_status | set_estimate | create_item | pull"`
	Path     string `json:"path,omitempty" jsonschema:"path of the item, when relevant"`
	Status   string `json:"status,omitempty" jsonschema:"target status when action=set_status"`
	Estimate string `json:"estimate,omitempty" jsonschema:"target estimate when action=set_estimate or create_item"`
	Type     string `json:"type,omitempty" jsonschema:"story type when action=create_item or coercing a check"`
}

// CoachVerdict is the structured response. allowed=false means the
// agent should refuse and surface the rule + next move. Source carries
// a canonical rule slug for callers that want to log it.
type CoachVerdict struct {
	Allowed bool   `json:"allowed"`
	Rule    string `json:"rule,omitempty"`
	Source  string `json:"source,omitempty"`
	Next    string `json:"next,omitempty"`
	Nudge   bool   `json:"nudge,omitempty" jsonschema:"true when the result is a soft warning, not a hard refusal"`
	Detail  string `json:"detail,omitempty"`
}

// coachCheckTool runs the canon checks. Hard refusals (allowed=false) on:
//   - 8-pt cap on features
//   - bugs/chores with an estimate
//   - dev-as-PM accepting their own story (always render PM ceremony)
//   - releases moving through state machine
//
// Nudges (allowed=true, nudge=true) on:
//   - feature start with no `## Acceptance` section in the body
//   - iteration overcommit beyond rolling velocity
func coachCheckTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
		switch strings.ToLower(strings.TrimSpace(args.Action)) {
		case "set_status":
			return checkSetStatus(root, args)
		case "set_estimate":
			return checkSetEstimate(root, args)
		case "create_item":
			return checkCreateItem(args)
		case "pull":
			return checkPull(root, args)
		}
		return nil, CoachVerdict{Allowed: true, Detail: "no rule registered for this action"}, nil
	}
}

// checkPull is the pre-pull alignment gate. It fires when the agent is
// about to start coding on a story (am pull, am-align skill, or any
// caller that wants the same hard check). When a feature has no
// `## Acceptance` section in its body, the check refuses: agents that
// confidently build the wrong thing are the central failure mode of
// AI-paired delivery, and a story without acceptance criteria gives the
// agent nothing to build toward.
func checkPull(root *backlog.BacklogsStructure, args CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
	if args.Path == "" {
		return nil, CoachVerdict{}, fmt.Errorf("path is required for pull check")
	}
	item, _, err := loadExistingItem(root, args.Path)
	if err != nil {
		return nil, CoachVerdict{}, err
	}
	// Acceptance bullets were removed from the model. The pull check
	// used to refuse a feature without a `## Acceptance` section here;
	// with the bullets gone, there's nothing to gate against, so pull
	// is always allowed. Item is still loaded above so a missing path
	// surfaces as an error.
	_ = item
	return nil, CoachVerdict{Allowed: true}, nil
}

func checkSetStatus(root *backlog.BacklogsStructure, args CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
	if args.Path == "" {
		return nil, CoachVerdict{}, fmt.Errorf("path is required for set_status check")
	}
	target := strings.ToLower(strings.TrimSpace(args.Status))
	item, _, err := loadExistingItem(root, args.Path)
	if err != nil {
		return nil, CoachVerdict{}, err
	}
	if target == backlog.AcceptedStatus.Name {
		// Autonomous mode collapses the dev-pair / PM seam: the agent may
		// accept its own work. The value shifts from gating each story to
		// capturing why it met the bar — so allow, but nudge to record a
		// rationale that lands in the audit trail (`am journal`).
		cfg, _ := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if cfg != nil && cfg.IsAutonomous() {
			return nil, CoachVerdict{
				Allowed: true,
				Nudge:   true,
				Rule:    "autonomous mode: dev pair may accept its own work",
				Source:  "autonomous-mode",
				Next:    fmt.Sprintf("record why it meets the bar so `am journal` captures it: am accept %s --note \"...\"", args.Path),
			}, nil
		}
		return nil, CoachVerdict{
			Allowed: false,
			Rule:    "the dev pair does not accept its own work",
			Source:  "coach-refuses-pm-accepts",
			Next:    "wait for the human to accept from the detail panel (or switch to autonomous mode: `am mode autonomous`)",
		}, nil
	}
	if item.Type() == "release" {
		return nil, CoachVerdict{
			Allowed: false,
			Rule:    "releases are date markers, not state-machine items",
			Source:  "dates-slide-scope-doesnt",
			Next:    "leave status alone; update release_date instead",
		}, nil
	}
	return nil, CoachVerdict{Allowed: true}, nil
}

func checkSetEstimate(root *backlog.BacklogsStructure, args CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
	if args.Path == "" {
		return nil, CoachVerdict{}, fmt.Errorf("path is required for set_estimate check")
	}
	pts, err := strconv.ParseFloat(strings.TrimSpace(args.Estimate), 64)
	if err != nil {
		return nil, CoachVerdict{Allowed: false, Rule: "estimate must be numeric", Source: "8-point-hard-cap"}, nil
	}
	item, _, err := loadExistingItem(root, args.Path)
	if err != nil {
		return nil, CoachVerdict{}, err
	}
	cfg, _ := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
	typ := item.Type()
	if typ == "" {
		typ = "feature"
	}
	if pts > 0 {
		if typ == "bug" && (cfg == nil || !cfg.StoryTypes.BugEstimable) {
			return nil, CoachVerdict{
				Allowed: false,
				Rule:    "bugs are not pointed by default",
				Source:  "bugs-are-tax",
				Next:    "strip the estimate, or convert to a feature, or set story_types.bug_estimable in .am/config.yaml",
			}, nil
		}
		if typ == "chore" && (cfg == nil || !cfg.StoryTypes.ChoreEstimable) {
			return nil, CoachVerdict{
				Allowed: false,
				Rule:    "chores are not pointed by default",
				Source:  "toil-is-not-progress",
				Next:    "strip the estimate, or set story_types.chore_estimable in .am/config.yaml",
			}, nil
		}
	}
	if typ == "feature" && pts > 8 {
		return nil, CoachVerdict{
			Allowed: false,
			Rule:    "features over 8 points are epics",
			Source:  "8-point-hard-cap",
			Next:    "split the story; keep each piece at or below 8 points",
		}, nil
	}
	return nil, CoachVerdict{Allowed: true}, nil
}

func checkCreateItem(args CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
	if args.Estimate == "" {
		return nil, CoachVerdict{Allowed: true}, nil
	}
	pts, err := strconv.ParseFloat(strings.TrimSpace(args.Estimate), 64)
	if err != nil {
		return nil, CoachVerdict{Allowed: false, Rule: "estimate must be numeric"}, nil
	}
	typ := strings.ToLower(strings.TrimSpace(args.Type))
	if typ == "" {
		typ = "feature"
	}
	if typ == "feature" && pts > 8 {
		return nil, CoachVerdict{
			Allowed: false,
			Rule:    "features over 8 points are epics",
			Source:  "8-point-hard-cap",
			Next:    "split the story before creating it",
		}, nil
	}
	return nil, CoachVerdict{Allowed: true}, nil
}

// IterationFitArgs reports whether a backlog's current iteration window
// would fit within the rolling velocity, optionally also assuming a
// candidate item is added.
type IterationFitArgs struct {
	Backlog       string `json:"backlog"`
	CandidatePath string `json:"candidate_path,omitempty" jsonschema:"optional path; if set, points are added to the planned total"`
}

type IterationFitResult struct {
	Velocity        float64 `json:"velocity"`
	IterationLength int     `json:"iteration_length_weeks"`
	Planned         float64 `json:"planned_points"`
	Fits            bool    `json:"fits"`
	Delta           float64 `json:"delta_points" jsonschema:"positive when planned exceeds velocity"`
	ItemsCounted    int     `json:"items_counted"`
}

func iterationFitTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, IterationFitArgs) (*mcp.CallToolResult, IterationFitResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args IterationFitArgs) (*mcp.CallToolResult, IterationFitResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, IterationFitResult{}, err
		}
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, IterationFitResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, IterationFitResult{}, err
		}
		now := time.Now().In(cfg.IterationLocation())
		velocity := computeVelocity(bck, cfg, root.Root())
		// Planned = sum of estimates of priority-list items in current
		// iteration band. We approximate the band as items whose accepted
		// timestamp is empty AND status != accepted (i.e. unfinished).
		// The board derives the same window client-side; this server-side
		// number is used by the coach for nudges.
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return nil, IterationFitResult{}, err
		}
		byPath := indexItems(bck)
		var planned float64
		var counted int
		cap := velocity
		for _, e := range pri.Entries() {
			it, ok := byPath[e.Path]
			if !ok {
				continue
			}
			if strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
				continue
			}
			pts := parsePoints(it.Estimate())
			if pts <= 0 {
				continue
			}
			if cap > 0 && planned+pts > cap {
				break
			}
			planned += pts
			counted++
		}
		if args.CandidatePath != "" {
			cand, _, err := loadExistingItem(root, args.CandidatePath)
			if err == nil {
				planned += parsePoints(cand.Estimate())
			}
		}
		_ = now
		fits := velocity <= 0 || planned <= velocity
		delta := planned - velocity
		return nil, IterationFitResult{
			Velocity:        velocity,
			IterationLength: cfg.Iteration.LengthWeeks,
			Planned:         planned,
			Fits:            fits,
			Delta:           delta,
			ItemsCounted:    counted,
		}, nil
	}
}
