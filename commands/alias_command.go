package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"github.com/urfave/cli/v3"
	"runtime"
)

var AliasCommand = &cli.Command{
	Name:      "alias",
	Usage:     "Add a Bash alias for the script",
	ArgsUsage: "ALIAS",
	Action: func(ctx context.Context, c *cli.Command) error {
		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
			fmt.Println("Unsupported OS")
			return nil
		}

		if c.NArg() != 1 {
			fmt.Println("an alias should be specified")
			return nil
		}

		err := autocomplete.AddAliasWithBashAutoComplete(c.Args().Get(0))
		if err != nil {
			fmt.Println(err)
			return nil
		}

		fmt.Println("Please, restart your terminal session to use the new alias")
		return nil
	},
}
