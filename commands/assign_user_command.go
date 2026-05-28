package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

// AssignUserCommand has two modes:
//   - `am assign -s STATUS_CODE`: interactive assigner (legacy).
//   - `am assign ITEM_PATH USER...`: non-interactive, accepts up to three
//     assignees. Single user writes a YAML scalar; two or more write a
//     flow sequence.
var AssignUserCommand = &cli.Command{
	Name:      "assign",
	Usage:     "Assign a story to one or more users",
	ArgsUsage: "ITEM_PATH USER... | -s STATUS",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("List items in this status, then assign them to a user. %s", backlog.AllStatusesList()),
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() >= 2 {
			path, err := filepath.Abs(c.Args().Get(0))
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}
			users := append([]string(nil), c.Args().Slice()[1:]...)
			if len(users) > 3 {
				return fmt.Errorf("at most 3 assignees allowed; got %d", len(users))
			}
			item, err := backlog.LoadBacklogItem(path)
			if err != nil {
				return err
			}
			item.SetAssignees(users)
			item.SetModified(utils.GetCurrentTimestamp())
			if err := item.Save(); err != nil {
				return err
			}
			fmt.Printf("%s -> assigned to %s\n", filepath.Base(path), strings.Join(users, ", "))
			return nil
		}

		statusCode := c.String("s")
		if statusCode == "" {
			fmt.Println("usage: am assign ITEM_PATH USER... (or -s STATUS for interactive mode)")
			return nil
		}
		if !backlog.IsValidStatusCode(statusCode) {
			fmt.Printf("illegal status: %s\n", statusCode)
			return nil
		}
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		backlogDir, _ := filepath.Abs(".")
		action := actions.NewAssignUserAction(backlogDir, statusCode)
		return action.Execute()
	},
}
