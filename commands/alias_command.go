package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"gopkg.in/urfave/cli.v1"
	"runtime"
)

var AliasCommand = cli.Command{
	Name:      "alias",
	Usage:     "Add a Bash alias for the script",
	ArgsUsage: "ALIAS",
	Action: func(c *cli.Context) error {
		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
			fmt.Println("Unsupported OS")
			return nil
		}

		if c.NArg() != 1 {
			fmt.Println("an alias should be specified")
			return nil
		}

		err := autocomplete.AddAliasWithBashAutoComplete(c.Args()[0])
		if err != nil {
			fmt.Println(err)
			return nil
		}

		fmt.Println("Please, restart your terminal session to use the new alias")
		return nil
	},
}
