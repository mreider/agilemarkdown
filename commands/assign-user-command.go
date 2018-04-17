package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
)

type AssignUserCommand struct {
	Status  string `short:"s" required:"true" description:"status (f, l, g or h)"`
	RootDir string
}

func (*AssignUserCommand) Name() string {
	return "assign"
}

func (cmd *AssignUserCommand) Execute(args []string) error {
	if cmd.Status != "f" && cmd.Status != "l" && cmd.Status != "g" && cmd.Status != "h" && cmd.Status != "all" {
		return fmt.Errorf("illegal status: %s", cmd.Status)
	}
	if err := checkIsBacklogDirectory(cmd.RootDir); err != nil {
		return err
	}
	bck, err := backlog.LoadBacklog(cmd.RootDir)
	if err != nil {
		return err
	}

	items := bck.ItemsByStatus(cmd.Status)
	// TODO
	for i, item := range items {
		fmt.Println(i+1, item.Title())
	}

	return nil
}
