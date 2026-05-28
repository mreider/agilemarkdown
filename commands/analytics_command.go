package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

// CycleTimeCommand prints cycle time stats for the current backlog.
var CycleTimeCommand = &cli.Command{
	Name:      "cycle-time",
	Usage:     "Show cycle time (started -> accepted) median and longest items",
	ArgsUsage: " ",
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		dir, err := filepath.Abs(".")
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		fmt.Print(backlog.CycleTimeASCII(bck))
		return nil
	},
}

// RejectionRateCommand prints rejection rate per iteration over the
// rolling lookback window.
var RejectionRateCommand = &cli.Command{
	Name:      "rejection-rate",
	Usage:     "Show per-iteration rejection rate over the rolling lookback window",
	ArgsUsage: " ",
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		dir, err := filepath.Abs(".")
		if err != nil {
			return err
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		fmt.Print(backlog.RejectionRateASCII(time.Now(), bck.AllItems(), cfg))
		return nil
	},
}
