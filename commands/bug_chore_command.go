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

// `am bug` and `am chore` collapse the three-step dance the Pivotal flow
// otherwise requires when something needs fixing or some untyped infra
// work appears:
//
//	am create-item "Bug: X" --target priority --position top
//	am set-type <slug>.md bug
//	# (and remember bugs/chores carry no estimate)
//
// into a single verb. Both create the item, set the type, strip any
// estimate (bugs and chores are never pointed), and drop it at the top
// of priority so it is the next thing pulled — which is exactly the
// "no fix without a bug, and the bug goes to the top" rule.
var BugCommand = newBugChoreCommand("bug", "Create a bug at the top of priority (no estimate) and ready to pull")
var ChoreCommand = newBugChoreCommand("chore", "Create a chore at the top of priority (no estimate) and ready to pull")

func newBugChoreCommand(itemType, usage string) *cli.Command {
	return &cli.Command{
		Name:      itemType,
		Usage:     usage,
		ArgsUsage: "TITLE",
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := checkIsBacklogDirectory(); err != nil {
				return err
			}
			if c.NArg() == 0 {
				return fmt.Errorf("a %s title should be specified", itemType)
			}
			title := strings.Join(c.Args().Slice(), " ")

			action := actions.NewCreateItemAction(".", title, "", false)
			if err := action.Execute(); err != nil {
				return err
			}

			dir, err := filepath.Abs(".")
			if err != nil {
				return err
			}
			fileName := utils.GetValidFileName(title) + ".md"
			path := filepath.Join(dir, fileName)

			// Type the story and strip any estimate: bugs and chores are
			// never pointed (the coach rule enforced elsewhere; we honor
			// it at creation so the item is born compliant).
			item, err := backlog.LoadBacklogItem(path)
			if err != nil {
				return err
			}
			item.SetType(itemType)
			item.SetEstimate("")
			item.SetModified(utils.GetCurrentTimestamp())
			if err := item.Save(); err != nil {
				return err
			}

			// Place at the top of priority, lifting it out of the icebox
			// if create-item parked it there.
			entry := backlog.OrderEntry{Title: title, Path: fileName}
			if ice, ierr := backlog.LoadIcebox(dir); ierr == nil && ice != nil && ice.IndexOf(fileName) >= 0 {
				ice.Remove(fileName)
				_ = ice.Save()
			}
			if pri, perr := backlog.LoadPriority(dir); perr == nil && pri != nil {
				if pri.IndexOf(fileName) < 0 {
					pri.InsertAt(0, entry)
					_ = pri.Save()
				}
			}

			recordActivity(itemType, path, "target=priority position=top")
			fmt.Printf("%s created: %s (type=%s, top of priority)\n", itemType, fileName, itemType)
			return nil
		},
	}
}
