package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// StrengthCommand sets per-iteration team-strength and length overrides
// in `.am/iterations.yaml`. Mirrors Pivotal Tracker's
// `iteration_override` API resource.
//
//	am strength NUMBER VALUE         # set strength on iteration NUMBER
//	am strength NUMBER 0             # exclude this iteration from velocity entirely
//	am strength NUMBER --length N    # also override iteration length
//	am strength NUMBER --unset       # remove the override record
//	am strength --list               # show all active overrides
var StrengthCommand = &cli.Command{
	Name:      "strength",
	Usage:     "Set or list per-iteration team-strength and length overrides",
	ArgsUsage: "NUMBER [VALUE]",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "length", Usage: "override iteration length in weeks"},
		&cli.BoolFlag{Name: "unset", Usage: "remove the override record for NUMBER"},
		&cli.BoolFlag{Name: "list", Usage: "list every active override"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		overrides, err := backlog.LoadIterationOverrides(root)
		if err != nil {
			return err
		}

		if c.Bool("list") {
			if len(overrides.Overrides) == 0 {
				fmt.Println("(no overrides)")
				return nil
			}
			for _, o := range overrides.Overrides {
				fmt.Printf("iteration %d  team_strength=%g  length_weeks=%d\n", o.Number, o.TeamStrength, o.LengthWeeks)
			}
			return nil
		}

		if c.NArg() < 1 {
			return fmt.Errorf("usage: am strength NUMBER VALUE  OR  --list  OR  NUMBER --unset")
		}
		num, err := strconv.Atoi(strings.TrimSpace(c.Args().Get(0)))
		if err != nil {
			return fmt.Errorf("NUMBER must be an integer: %w", err)
		}

		if c.Bool("unset") {
			overrides.Clear(num)
			if err := overrides.Save(root); err != nil {
				return err
			}
			fmt.Printf("iteration %d override cleared\n", num)
			return nil
		}

		strength := -1.0 // sentinel: leave unchanged
		if c.NArg() >= 2 {
			s, err := strconv.ParseFloat(strings.TrimSpace(c.Args().Get(1)), 64)
			if err != nil {
				return fmt.Errorf("VALUE must be a number: %w", err)
			}
			if s < 0 {
				return fmt.Errorf("strength must be >= 0 (0 excludes the iteration)")
			}
			strength = s
		}
		length := c.Int("length")
		if strength < 0 && length <= 0 {
			return fmt.Errorf("provide VALUE or --length")
		}
		overrides.Set(num, strength, length)
		if err := overrides.Save(root); err != nil {
			return err
		}
		rec := overrides.Find(num)
		fmt.Printf("iteration %d  team_strength=%g  length_weeks=%d\n", num, rec.TeamStrength, rec.LengthWeeks)
		return nil
	},
}
