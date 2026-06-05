package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

// ModeCommand reads or sets the project's workflow mode in
// .am/config.yaml. With no argument it prints the effective mode (and
// flags when AM_MODE is overriding the configured value); with an
// argument it persists the new mode.
//
//	am mode               # show
//	am mode autonomous    # let the dev pair run the full loop incl. accept
//	am mode coached       # restore the Pivotal dev-pair / PM seam (default)
var ModeCommand = &cli.Command{
	Name:      "mode",
	Usage:     "Show or set the workflow mode: coached (default) or autonomous",
	ArgsUsage: "[coached|autonomous]",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		cfgPath := filepath.Join(root, ".am", "config.yaml")
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return err
		}

		// Read.
		if c.NArg() == 0 {
			fmt.Printf("mode: %s\n", cfg.EffectiveMode())
			if env := strings.ToLower(strings.TrimSpace(os.Getenv("AM_MODE"))); env == config.ModeCoached || env == config.ModeAutonomous {
				fmt.Printf("  (AM_MODE=%s overriding configured mode %q)\n", env, cfg.Mode)
			}
			if cfg.EffectiveMode() == config.ModeAutonomous {
				fmt.Println("  autonomous: the dev pair may accept its own work; record why with `am accept ITEM --note \"...\"` and review with `am journal`.")
			} else {
				fmt.Println("  coached: stage work at delivered and wait for the PM to accept.")
			}
			return nil
		}

		// Write.
		want := strings.ToLower(strings.TrimSpace(c.Args().Get(0)))
		if want != config.ModeCoached && want != config.ModeAutonomous {
			return fmt.Errorf("mode must be %s or %s, got %q", config.ModeCoached, config.ModeAutonomous, want)
		}
		cfg.Mode = want
		if err := cfg.Save(cfgPath); err != nil {
			return err
		}
		recordActivity("mode", "", want)
		fmt.Printf("mode -> %s\n", want)
		env := strings.ToLower(strings.TrimSpace(os.Getenv("AM_MODE")))
		if (env == config.ModeCoached || env == config.ModeAutonomous) && env != want {
			fmt.Printf("  note: AM_MODE=%s is still set and overrides this until unset.\n", env)
		}
		return nil
	},
}
