package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/mreider/agilemarkdown/commands"
	"os"
)

func main() {
	var createBacklogCmd = &commands.CreateBacklogCommand{"."}
	var createItemCmd = &commands.CreateItemCommand{"."}
	var assignUserCmd = &commands.AssignUserCommand{RootDir: "."}

	parser := flags.NewParser(nil, flags.Default)
	parser.AddCommand(createBacklogCmd.Name(), "Create a new backlog", "", createBacklogCmd)
	parser.AddCommand(createItemCmd.Name(), "Create a new item for the backlog", "", createItemCmd)
	parser.AddCommand(assignUserCmd.Name(), "Assign a story to a user", "", assignUserCmd)

	_, err := parser.Parse()
	if err != nil {
		os.Exit(-1)
	}

}
