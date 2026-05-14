package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

// loadItemFromArg resolves an item path argument (relative or absolute,
// .md optional) into a loaded BacklogItem.
func loadItemFromArg(arg string) (*backlog.BacklogItem, error) {
	if arg == "" {
		return nil, fmt.Errorf("item path required")
	}
	if !strings.HasSuffix(arg, ".md") {
		arg += ".md"
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return nil, err
	}
	return backlog.LoadBacklogItem(abs)
}

// EstimateCommand sets the story-point estimate on an item.
//
// With --advise, prints a Pivotal-style framing for picking a value
// (Fibonacci by uncertainty, not effort) plus a quick reference, then
// exits without writing. Useful for solo CLI users who do not have
// the am-decompose skill.
var EstimateCommand = &cli.Command{
	Name:      "estimate",
	Usage:     "Set the story-point estimate on an item; --advise prints the framing without writing",
	ArgsUsage: "ITEM_PATH POINTS",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "advise", Usage: "print the Pivotal estimation framing and exit"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.Bool("advise") {
			printEstimationAdvice()
			return nil
		}
		if c.NArg() != 2 {
			return fmt.Errorf("usage: am estimate ITEM_PATH POINTS  (or am estimate --advise)")
		}
		item, err := loadItemFromArg(c.Args().Get(0))
		if err != nil {
			return err
		}
		item.SetEstimate(strings.TrimSpace(c.Args().Get(1)))
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s estimate -> %s\n", filepath.Base(item.Path()), item.Estimate())
		return nil
	},
}

// printEstimationAdvice prints a quick Pivotal-style framing for
// picking a story-point value. The aim is to keep solo users honest
// when the am-decompose skill is not running for them.
func printEstimationAdvice() {
	fmt.Println("Pivotal estimation, in one screen:")
	fmt.Println()
	fmt.Println("  Point by uncertainty, not by effort. The number answers")
	fmt.Println("  'how confident am I that this story can be done in one")
	fmt.Println("  iteration?' rather than 'how many hours will it take?'")
	fmt.Println()
	fmt.Println("  Fibonacci scale, with a hard cap at 8:")
	fmt.Println("    1   small, well-trodden change. Confidence is high.")
	fmt.Println("    2   small but new in some way. Mostly known.")
	fmt.Println("    3   moderate. Some unknowns; you can still see the shape.")
	fmt.Println("    5   large. Real unknowns; the team can name them.")
	fmt.Println("    8   the hard cap. Above this is an epic; split it.")
	fmt.Println()
	fmt.Println("  Bugs and chores are not pointed by default. They flow as")
	fmt.Println("  tax (bugs) or toil (chores), and the velocity ledger does")
	fmt.Println("  not credit them.")
	fmt.Println()
	fmt.Println("  Three-point red flag: a 3 that the team is not confident")
	fmt.Println("  about is usually a 5 in disguise.")
	fmt.Println()
	fmt.Println("Set the estimate with: am estimate ITEM_PATH POINTS")
}

// TagCommand sets, adds, or removes tags on an item.
var TagCommand = &cli.Command{
	Name:      "tag",
	Usage:     "Set tags on an item (replace), or with --add/--remove modify in place",
	ArgsUsage: "ITEM_PATH [TAG ...]",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{Name: "add", Usage: "add this tag (repeatable)"},
		&cli.StringSliceFlag{Name: "remove", Usage: "remove this tag (repeatable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return fmt.Errorf("usage: am tag ITEM_PATH [TAG ...]  OR  am tag ITEM_PATH --add T --remove T")
		}
		item, err := loadItemFromArg(c.Args().Get(0))
		if err != nil {
			return err
		}

		add := c.StringSlice("add")
		remove := c.StringSlice("remove")
		positional := c.Args().Slice()[1:]

		var next []string
		switch {
		case len(positional) > 0 && (len(add) > 0 || len(remove) > 0):
			return fmt.Errorf("either pass tags positionally to replace, or use --add/--remove, not both")
		case len(positional) > 0:
			next = positional
		default:
			next = item.Tags()
			for _, t := range remove {
				next = removeTag(next, t)
			}
			for _, t := range add {
				if !containsTag(next, t) {
					next = append(next, t)
				}
			}
		}
		item.SetTags(next)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s tags -> [%s]\n", filepath.Base(item.Path()), strings.Join(item.Tags(), ", "))
		return nil
	},
}

// EpicCommand sets the epic slug on an item, or clears it with --unset.
var EpicCommand = &cli.Command{
	Name:      "epic",
	Usage:     "Set the epic slug on an item, or --unset to clear",
	ArgsUsage: "ITEM_PATH [SLUG]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "unset", Usage: "clear the epic field"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return fmt.Errorf("usage: am epic ITEM_PATH SLUG  OR  am epic ITEM_PATH --unset")
		}
		item, err := loadItemFromArg(c.Args().Get(0))
		if err != nil {
			return err
		}
		if c.Bool("unset") {
			item.SetEpic("")
		} else {
			if c.NArg() != 2 {
				return fmt.Errorf("usage: am epic ITEM_PATH SLUG")
			}
			item.SetEpic(c.Args().Get(1))
		}
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		if item.Epic() == "" {
			fmt.Printf("%s epic -> (cleared)\n", filepath.Base(item.Path()))
		} else {
			fmt.Printf("%s epic -> %s\n", filepath.Base(item.Path()), item.Epic())
		}
		return nil
	},
}

// HypothesisCommand sets the legacy `hypothesis:` frontmatter on an item.
// The Pivotal way uses acceptance criteria under `## Acceptance` in the
// body, not a separate hypothesis field. This command is kept for
// back-compat and prints a deprecation banner before acting.
var HypothesisCommand = &cli.Command{
	Name:      "hypothesis",
	Usage:     "Deprecated: set the legacy hypothesis frontmatter (prefer acceptance criteria in the body)",
	ArgsUsage: "ITEM_PATH \"text\"",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "unset", Usage: "clear the hypothesis field"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		fmt.Fprintln(os.Stderr, "warning: am hypothesis is deprecated. Pivotal acceptance lives in the body's `## Acceptance` section.")
		if c.NArg() < 1 {
			return fmt.Errorf("usage: am hypothesis ITEM_PATH \"text\"")
		}
		item, err := loadItemFromArg(c.Args().Get(0))
		if err != nil {
			return err
		}
		if c.Bool("unset") {
			item.SetHypothesis("")
		} else {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: am hypothesis ITEM_PATH \"text\"")
			}
			text := strings.Join(c.Args().Slice()[1:], " ")
			item.SetHypothesis(text)
		}
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s hypothesis updated\n", filepath.Base(item.Path()))
		return nil
	},
}

func containsTag(tags []string, t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	for _, x := range tags {
		if strings.EqualFold(strings.TrimSpace(x), t) {
			return true
		}
	}
	return false
}

func removeTag(tags []string, t string) []string {
	out := make([]string, 0, len(tags))
	for _, x := range tags {
		if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(t)) {
			continue
		}
		out = append(out, x)
	}
	return out
}
