package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/mreider/agilemarkdown/commands"
	"os"
)

func main() {
	var createBacklogCmd = &commands.CreateBacklogCommand{RootDir: "."}
	var createItemCmd = &commands.CreateItemCommand{RootDir: "."}
	var assignUserCmd = &commands.AssignUserCommand{RootDir: "."}
	var changeStatusCmd = &commands.ChangeStatusCommand{RootDir: "."}
	var workCmd = &commands.WorkCommand{RootDir: "."}
	var pointsCmd = &commands.PointsCommand{RootDir: "."}
	var syncCmd = &commands.SyncCommand{RootDir: "."}

	parser := flags.NewParser(nil, flags.Default)
	parser.AddCommand(createBacklogCmd.Name(), "Create a new backlog", "", createBacklogCmd)
	parser.AddCommand(createItemCmd.Name(), "Create a new item for the backlog", "", createItemCmd)
	parser.AddCommand(assignUserCmd.Name(), "Assign a story to a user", "", assignUserCmd)
	parser.AddCommand(changeStatusCmd.Name(), "Change story status", "", changeStatusCmd)
	parser.AddCommand(workCmd.Name(), "Show user work by status", "", workCmd)
	parser.AddCommand(pointsCmd.Name(), "Show total points by user and status", "", pointsCmd)
	parser.AddCommand(syncCmd.Name(), "Sync state", "", syncCmd)

	_, err := parser.Parse()
	if err != nil {
		os.Exit(-1)
	}

}
