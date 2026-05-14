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
	path := filepath.Join(root.Root(), args.Path)
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return nil, CoachVerdict{}, err
	}
	typ := item.Type()
	if typ == "" {
		typ = "feature"
	}
	// Only features need acceptance criteria. Bugs and chores have their
	// own conventions for "done" that do not flow through the bullet
	// ceremony.
	if typ != "feature" {
		return nil, CoachVerdict{Allowed: true}, nil
	}
	if backlog.AcceptanceBulletTexts(item.Body()) == nil {
		return nil, CoachVerdict{
			Allowed: false,
			Rule:    "feature has no acceptance criteria",
			Source:  "acceptance-before-pull",
			Next:    fmt.Sprintf("draft a `## Acceptance` section in %s, or run /am-align before pulling", args.Path),
		}, nil
	}
	return nil, CoachVerdict{Allowed: true}, nil
}

func checkSetStatus(root *backlog.BacklogsStructure, args CoachCheckArgs) (*mcp.CallToolResult, CoachVerdict, error) {
	if args.Path == "" {
		return nil, CoachVerdict{}, fmt.Errorf("path is required for set_status check")
	}
	target := strings.ToLower(strings.TrimSpace(args.Status))
	path := filepath.Join(root.Root(), args.Path)
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return nil, CoachVerdict{}, err
	}
	if target == backlog.AcceptedStatus.Name {
		return nil, CoachVerdict{
			Allowed: false,
			Rule:    "the dev pair does not accept its own work",
			Source:  "coach-refuses-pm-accepts",
			Next:    fmt.Sprintf("render PM ceremony with acceptance_prompt(path=%q) and wait for the human", args.Path),
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
	if target == backlog.StartedStatus.Name && (item.Type() == "" || item.Type() == "feature") {
		if backlog.AcceptanceBulletTexts(item.Body()) == nil {
			return nil, CoachVerdict{
				Allowed: true,
				Nudge:   true,
				Rule:    "feature has no acceptance criteria",
				Source:  "acceptance-criteria",
				Next:    "add a `## Acceptance` section with bullets to the body before starting, or skip with intent",
			}, nil
		}
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
	path := filepath.Join(root.Root(), args.Path)
	item, err := backlog.LoadBacklogItem(path)
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

// AcceptancePromptArgs renders the PM ceremony for one delivered story.
type AcceptancePromptArgs struct {
	Path string `json:"path"`
}

type AcceptancePromptResult struct {
	Path       string                `json:"path"`
	Title      string                `json:"title"`
	Type       string                `json:"type"`
	Status     string                `json:"status"`
	Estimate   string                `json:"estimate,omitempty"`
	Verify     []string              `json:"verify,omitempty" jsonschema:"bullet text only; preserved for clients that cached the old shape"`
	Bullets    []AcceptanceBulletRow `json:"bullets,omitempty" jsonschema:"each bullet with index, state (open/claimed/verified), text, optional claim_note"`
	PromptText string                `json:"prompt_text"`
}

// acceptancePromptTool returns the structured ceremony body. The agent
// renders it to the human and waits for an answer; the answer drives
// set_status or reject_item.
func acceptancePromptTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, AcceptancePromptArgs) (*mcp.CallToolResult, AcceptancePromptResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args AcceptancePromptArgs) (*mcp.CallToolResult, AcceptancePromptResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, AcceptancePromptResult{}, err
		}
		bullets := backlog.ParseAcceptance(item.Body())
		verify := make([]string, 0, len(bullets))
		for _, b := range bullets {
			verify = append(verify, b.Text)
		}
		rows := bulletsToRows(bullets)
		typ := item.Type()
		if typ == "" {
			typ = "feature"
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Story: %s (%s)\n", item.Title(), args.Path)
		fmt.Fprintf(&b, "Type: %s\n", typ)
		fmt.Fprintf(&b, "Status staging: %s -> accepted\n", item.Status())
		if est := item.Estimate(); est != "" {
			fmt.Fprintf(&b, "Estimate: %s pts\n", est)
		}
		if len(bullets) > 0 {
			b.WriteString("What to verify:\n")
			for _, bl := range bullets {
				marker := bulletMarker(bl.State)
				fmt.Fprintf(&b, "  %s %s\n", marker, bl.Text)
				if bl.ClaimNote != "" {
					fmt.Fprintf(&b, "      (claim: %s)\n", bl.ClaimNote)
				}
			}
		} else {
			b.WriteString("What to verify: (no `## Acceptance` section in body; review the diff)\n")
		}
		b.WriteString("\nAs PM, do you accept?")

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: b.String()}},
		}, AcceptancePromptResult{
			Path:       args.Path,
			Title:      item.Title(),
			Type:       typ,
			Status:     item.Status(),
			Estimate:   item.Estimate(),
			Verify:     verify,
			Bullets:    rows,
			PromptText: b.String(),
		}, nil
	}
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
			candPath := filepath.Join(root.Root(), args.CandidatePath)
			cand, err := backlog.LoadBacklogItem(candPath)
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
