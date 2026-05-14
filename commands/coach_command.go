package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

// AcceptancePromptCommand mirrors the acceptance_prompt MCP tool. Renders
// the PM ceremony for one delivered story so the human can review and
// answer before flipping status.
var AcceptancePromptCommand = &cli.Command{
	Name:      "accept-prompt",
	Usage:     "Render the PM acceptance ceremony for a delivered story",
	ArgsUsage: "ITEM_PATH",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am accept-prompt ITEM_PATH")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		path, err := filepath.Abs(c.Args().Get(0))
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			path += ".md"
		}
		rel, _ := filepath.Rel(root, path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		typ := item.Type()
		if typ == "" {
			typ = "feature"
		}
		fmt.Printf("Story: %s (%s)\n", item.Title(), rel)
		fmt.Printf("Type: %s\n", typ)
		fmt.Printf("Status staging: %s -> accepted\n", item.Status())
		if est := item.Estimate(); est != "" {
			fmt.Printf("Estimate: %s pts\n", est)
		}
		bullets := backlog.AcceptanceBulletTexts(item.Body())
		if len(bullets) > 0 {
			fmt.Println("What to verify:")
			for _, b := range bullets {
				fmt.Printf("  - %s\n", b)
			}
		} else {
			fmt.Println("What to verify: (no `## Acceptance` section in body; review the diff)")
		}
		fmt.Println()
		fmt.Println("As PM, do you accept?")
		fmt.Println("  yes:  am accept " + rel)
		fmt.Println("  no:   am reject " + rel + " --reason \"...\"")
		return nil
	},
}

// CoachCheckCommand mirrors the coach_check MCP tool. Useful when a
// non-MCP scripting layer wants to preflight an action.
//
// The action can be passed positionally (`am coach-check set_status`)
// or via the `--action` flag (`am coach-check --action set_status`).
// Both forms are equivalent; the flag form matches the natural reading
// of the coach doc, the positional form matches typical CLI shape.
var CoachCheckCommand = &cli.Command{
	Name:      "coach-check",
	Usage:     "Preflight an action against Pivotal canon (set_status / set_estimate / create_item)",
	ArgsUsage: "ACTION",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "action", Usage: "action to preflight (synonym for the positional ACTION argument)"},
		&cli.StringFlag{Name: "path", Usage: "item path"},
		&cli.StringFlag{Name: "status", Usage: "target status (with set_status)"},
		&cli.StringFlag{Name: "estimate", Usage: "target estimate (with set_estimate or create_item)"},
		&cli.StringFlag{Name: "type", Usage: "story type (with create_item)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		action := strings.TrimSpace(c.String("action"))
		if action == "" && c.NArg() == 1 {
			action = c.Args().Get(0)
		}
		if action == "" {
			return fmt.Errorf("usage: am coach-check ACTION [--path P] [--status S] [--estimate N] [--type T]\n   or: am coach-check --action ACTION [--path P] ...")
		}
		verdict, err := runCoachCheck(action, c.String("path"), c.String("status"), c.String("estimate"), c.String("type"))
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(verdict, "", "  ")
		fmt.Println(string(b))
		if !verdict.Allowed {
			return cli.Exit("", 1)
		}
		return nil
	},
}

// IterationFitCommand mirrors the iteration_fit MCP tool. Reports
// whether planned points fit the rolling velocity.
var IterationFitCommand = &cli.Command{
	Name:      "iteration-fit",
	Usage:     "Report whether the current iteration fits within rolling velocity",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "candidate", Usage: "optional path of a candidate item to add to planned total"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		dir, err := filepath.Abs(".")
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(dir)
		_ = bck
		if err != nil {
			return err
		}
		// Reuse the math via a thin re-exec of the MCP tool's logic. To
		// avoid duplicating it, the CLI prints a compact summary using
		// the same primitives directly.
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		velocity, planned, counted, err := iterationFitNumbers(root, dir, c.String("candidate"))
		if err != nil {
			return err
		}
		fits := velocity <= 0 || planned <= velocity
		fmt.Printf("velocity:    %.0f pts\n", velocity)
		fmt.Printf("planned:     %.0f pts (across %d items)\n", planned, counted)
		fmt.Printf("delta:       %+.0f pts\n", planned-velocity)
		if fits {
			fmt.Println("status:      fits")
		} else {
			fmt.Println("status:      OVER (working agreement: trim or unice)")
		}
		return nil
	},
}

type coachVerdict struct {
	Allowed bool   `json:"allowed"`
	Rule    string `json:"rule,omitempty"`
	Source  string `json:"source,omitempty"`
	Next    string `json:"next,omitempty"`
	Nudge   bool   `json:"nudge,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

func runCoachCheck(action, path, status, estimate, typ string) (coachVerdict, error) {
	// Light duplication of the server-side logic so the CLI doesn't
	// require an MCP session. Keep behavior in lock-step with
	// mcpserver/coach.go.
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "set_status":
		if path == "" {
			return coachVerdict{}, fmt.Errorf("path is required")
		}
		if strings.EqualFold(status, "accepted") {
			return coachVerdict{
				Allowed: false,
				Rule:    "the dev pair does not accept its own work",
				Source:  "coach-refuses-pm-accepts",
				Next:    fmt.Sprintf("render PM ceremony with `am accept-prompt %s`", path),
			}, nil
		}
	case "set_estimate":
		// Light client-side preflight; full logic is server-side. Just
		// catch the 8-pt cap and report. CLI users wanting full canon
		// should call the MCP tool.
		if estimate == "" {
			return coachVerdict{Allowed: true}, nil
		}
		var pts float64
		fmt.Sscanf(estimate, "%f", &pts)
		if pts > 8 {
			return coachVerdict{
				Allowed: false,
				Rule:    "features over 8 points are epics",
				Source:  "8-point-hard-cap",
				Next:    "split the story",
			}, nil
		}
	case "create_item":
		if estimate != "" {
			var pts float64
			fmt.Sscanf(estimate, "%f", &pts)
			if pts > 8 && (typ == "" || typ == "feature") {
				return coachVerdict{
					Allowed: false,
					Rule:    "features over 8 points are epics",
					Source:  "8-point-hard-cap",
					Next:    "split the story before creating it",
				}, nil
			}
		}
	case "pull":
		if path == "" {
			return coachVerdict{}, fmt.Errorf("path is required")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return coachVerdict{}, err
		}
		if !strings.HasSuffix(abs, ".md") {
			abs += ".md"
		}
		item, err := backlog.LoadBacklogItem(abs)
		if err != nil {
			return coachVerdict{}, err
		}
		itemType := item.Type()
		if itemType == "" {
			itemType = "feature"
		}
		if itemType == "feature" && backlog.AcceptanceBulletTexts(item.Body()) == nil {
			return coachVerdict{
				Allowed: false,
				Rule:    "feature has no acceptance criteria",
				Source:  "acceptance-before-pull",
				Next:    fmt.Sprintf("draft a `## Acceptance` section in %s, or run `am align %s` before pulling", filepath.Base(path), filepath.Base(path)),
			}, nil
		}
	}
	return coachVerdict{Allowed: true}, nil
}

func iterationFitNumbers(rootDir, backlogDir, candidate string) (velocity, planned float64, counted int, err error) {
	cfgPath := filepath.Join(rootDir, ".am", "config.yaml")
	bck, err := backlog.LoadBacklog(backlogDir)
	if err != nil {
		return 0, 0, 0, err
	}
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return 0, 0, 0, err
	}
	var feed []*backlog.BacklogItem
	for _, it := range bck.AllItems() {
		if backlog.CountsForVelocity(it, cfg) {
			feed = append(feed, it)
		}
	}
	overrides, _ := backlog.LoadIterationOverrides(rootDir)
	v, _, _ := backlog.ComputeVelocity(time.Now(), feed, cfg, overrides)
	pri, err := backlog.LoadPriority(backlogDir)
	if err != nil {
		return 0, 0, 0, err
	}
	byPath := map[string]*backlog.BacklogItem{}
	for _, it := range bck.ActiveItems() {
		byPath[filepath.Base(it.Path())] = it
	}
	cap := v
	for _, e := range pri.Entries() {
		it, ok := byPath[e.Path]
		if !ok || strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
			continue
		}
		var pts float64
		fmt.Sscanf(it.Estimate(), "%f", &pts)
		if pts <= 0 {
			continue
		}
		if cap > 0 && planned+pts > cap {
			break
		}
		planned += pts
		counted++
	}
	if candidate != "" {
		path := candidate
		if !strings.HasSuffix(path, ".md") {
			path += ".md"
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(rootDir, path)
		}
		cand, err := backlog.LoadBacklogItem(path)
		if err == nil {
			var pts float64
			fmt.Sscanf(cand.Estimate(), "%f", &pts)
			planned += pts
		}
	}
	return v, planned, counted, nil
}

