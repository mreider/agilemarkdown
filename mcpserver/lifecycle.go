package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const inceptionFileName = "inception.md"

// InceptionDocArgs reads or writes the project inception document.
//
// When body is empty, the tool reads the existing inception.md (or
// returns the default template if missing). When body is non-empty, the
// tool writes the body to inception.md, creating the file when needed.
type InceptionDocArgs struct {
	Body string `json:"body,omitempty" jsonschema:"if set, writes this body to inception.md; otherwise reads"`
}

type InceptionDocResult struct {
	Path    string `json:"path"`
	Body    string `json:"body"`
	Existed bool   `json:"existed"`
	Wrote   bool   `json:"wrote"`
}

func inceptionDocTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, InceptionDocArgs) (*mcp.CallToolResult, InceptionDocResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args InceptionDocArgs) (*mcp.CallToolResult, InceptionDocResult, error) {
		path := filepath.Join(root.Root(), inceptionFileName)
		_, statErr := os.Stat(path)
		existed := statErr == nil
		if strings.TrimSpace(args.Body) == "" {
			if !existed {
				return nil, InceptionDocResult{Path: inceptionFileName, Body: defaultInceptionBody, Existed: false}, nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, InceptionDocResult{}, err
			}
			return nil, InceptionDocResult{Path: inceptionFileName, Body: string(data), Existed: true}, nil
		}
		if err := os.WriteFile(path, []byte(args.Body), 0644); err != nil {
			return nil, InceptionDocResult{}, err
		}
		return nil, InceptionDocResult{Path: inceptionFileName, Body: args.Body, Existed: existed, Wrote: true}, nil
	}
}

// SprintPlanArgs renders the iteration plan for one backlog: the top of
// _priority.md up to rolling velocity, with flags on stories that need
// attention before the iteration starts.
type SprintPlanArgs struct {
	Backlog string `json:"backlog"`
}

type SprintPlanRow struct {
	Index             int      `json:"index"`
	Path              string   `json:"path"`
	Title             string   `json:"title"`
	Type              string   `json:"type"`
	Estimate          string   `json:"estimate"`
	Status            string   `json:"status"`
	Assignees         []string `json:"assignees,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Blocked           bool     `json:"blocked,omitempty"`
	HasAcceptance     bool     `json:"has_acceptance"`
	AcceptanceCount   int      `json:"acceptance_count"`
	OversizedFeature  bool     `json:"oversized_feature,omitempty"`
	UnestimatedFeature bool    `json:"unestimated_feature,omitempty"`
}

type SprintPlanResult struct {
	Backlog        string          `json:"backlog"`
	Velocity       float64         `json:"velocity"`
	IterationLength int            `json:"iteration_length_weeks"`
	Committed      []SprintPlanRow `json:"committed"`
	BelowLine      []SprintPlanRow `json:"below_line"`
	CommittedPts   float64         `json:"committed_points"`
	Overcommitted  bool            `json:"overcommitted"`
	Warnings       []string        `json:"warnings"`
}

func sprintPlanTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SprintPlanArgs) (*mcp.CallToolResult, SprintPlanResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SprintPlanArgs) (*mcp.CallToolResult, SprintPlanResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, SprintPlanResult{}, err
		}
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, SprintPlanResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, SprintPlanResult{}, err
		}
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return nil, SprintPlanResult{}, err
		}
		velocity := computeVelocity(bck, cfg, root.Root())
		byPath := indexItems(bck)

		committed := make([]SprintPlanRow, 0)
		below := make([]SprintPlanRow, 0)
		warnings := make([]string, 0)
		var pts float64

		cap := velocity
		for i, e := range pri.Entries() {
			it, ok := byPath[e.Path]
			if !ok {
				continue
			}
			row := buildSprintRow(i, it, e.Path)
			if !row.HasAcceptance && (it.Type() == "" || it.Type() == "feature") {
				warnings = append(warnings, fmt.Sprintf("%s: no `## Acceptance` section", e.Path))
			}
			if row.OversizedFeature {
				warnings = append(warnings, fmt.Sprintf("%s: feature over 8-point cap", e.Path))
			}
			if row.UnestimatedFeature {
				warnings = append(warnings, fmt.Sprintf("%s: feature with no estimate", e.Path))
			}
			p := parsePoints(it.Estimate())
			if cap > 0 && pts+p > cap {
				below = append(below, row)
				continue
			}
			pts += p
			committed = append(committed, row)
		}
		overcommit := velocity > 0 && pts > velocity
		return nil, SprintPlanResult{
			Backlog:         args.Backlog,
			Velocity:        velocity,
			IterationLength: cfg.Iteration.LengthWeeks,
			Committed:       committed,
			BelowLine:       below,
			CommittedPts:    pts,
			Overcommitted:   overcommit,
			Warnings:        warnings,
		}, nil
	}
}

func buildSprintRow(idx int, it *backlog.BacklogItem, basename string) SprintPlanRow {
	bullets := backlog.AcceptanceBulletTexts(it.Body())
	row := SprintPlanRow{
		Index:           idx,
		Path:            basename,
		Title:           it.Title(),
		Type:            it.Type(),
		Estimate:        it.Estimate(),
		Status:          it.Status(),
		Assignees:       it.Assignees(),
		Tags:            it.Tags(),
		Blocked:         it.Blocked(),
		HasAcceptance:   bullets != nil,
		AcceptanceCount: len(bullets),
	}
	pts, perr := strconv.ParseFloat(strings.TrimSpace(it.Estimate()), 64)
	typ := it.Type()
	if typ == "" {
		typ = "feature"
	}
	if typ == "feature" {
		if perr == nil && pts > 8 {
			row.OversizedFeature = true
		}
		if perr != nil || pts <= 0 {
			if it.Type() == "" || it.Type() == "feature" {
				row.UnestimatedFeature = true
			}
		}
	}
	return row
}

const defaultInceptionBody = `# Inception

## The user

Who, specifically, are we building this for? Real users with real circumstances, not a demographic.

## The goal

What changes for the user when we ship? Concrete, observable.

## The reason

Why this, why now? What changes if we do not?

## Success

How will we know it worked? The smallest signal that would tell the team the project did its job.

## Constraints

What can't move? Budget, deadline, dependencies, regulatory.

## Out of scope

What are we explicitly not doing?
`
