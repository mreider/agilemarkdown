package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/urfave/cli/v3"
)

var VelocityCommand = &cli.Command{
	Name:      "velocity",
	Usage:     "Show velocity per iteration. Default ASCII; --json emits structured rows.",
	ArgsUsage: "[NUMBER_OF_ITERATIONS]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit structured rows (iteration, planned, accepted, length_weeks, team_strength)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		var iterCount int
		if c.NArg() > 0 {
			iterCount, _ = strconv.Atoi(c.Args().Get(0))
		}

		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err != nil {
			return err
		}
		if c.Bool("json") {
			dir, derr := filepath.Abs(".")
			if derr != nil {
				return derr
			}
			res, err := mcpserver.VelocityHistory(ctx, root, mcpserver.VelocityHistoryArgs{
				Backlog:        filepath.Base(dir),
				IterationCount: iterCount,
			})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		bck, err := backlog.LoadBacklog(".")
		if err != nil {
			return err
		}
		overrides, _ := backlog.LoadIterationOverrides(root)
		if iterCount <= 0 {
			iterCount = 12
		}
		fmt.Print(backlog.VelocityASCII(bck, iterCount, cfg, overrides))
		return nil
	},
}
