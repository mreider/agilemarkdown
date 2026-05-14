package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

var TaskCommand = &cli.Command{
	Name:  "task",
	Usage: "Manage checkbox tasks under '## Tasks' on an item",
	Commands: []*cli.Command{
		{
			Name:      "list",
			Usage:     "List tasks for an item",
			ArgsUsage: "ITEM_PATH",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "json", Usage: "emit the task list as JSON (machine-readable)"},
			},
			Action: taskListAction,
		},
		{
			Name:      "add",
			Usage:     "Append a task to an item",
			ArgsUsage: "ITEM_PATH TEXT",
			Action:    taskAddAction,
		},
		{
			Name:      "tick",
			Usage:     "Toggle a task's checkbox by 1-based index",
			ArgsUsage: "ITEM_PATH INDEX",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "undo", Usage: "untick instead of tick"},
			},
			Action: taskTickAction,
		},
	},
}

func taskListAction(ctx context.Context, c *cli.Command) error {
	if c.NArg() != 1 {
		fmt.Println("usage: am task list ITEM_PATH")
		return nil
	}
	path := mustItemPath(c.Args().Get(0))
	if c.Bool("json") {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		res, err := mcpserver.ListTasks(ctx, root, mcpserver.ListTasksArgs{Path: rel})
		if err != nil {
			return err
		}
		return emitJSON(res)
	}
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	tasks := backlog.ParseTasks(item.Body())
	if len(tasks) == 0 {
		fmt.Println("(no tasks)")
		return nil
	}
	for _, t := range tasks {
		mark := " "
		if t.Done {
			mark = "x"
		}
		fmt.Printf("%2d. [%s] %s\n", t.Index, mark, t.Text)
	}
	return nil
}

func taskAddAction(ctx context.Context, c *cli.Command) error {
	if c.NArg() < 2 {
		fmt.Println("usage: am task add ITEM_PATH TEXT")
		return nil
	}
	path := mustItemPath(c.Args().Get(0))
	text := strings.TrimSpace(strings.Join(c.Args().Tail(), " "))
	if text == "" {
		fmt.Println("task text is required")
		return nil
	}
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	item.SetBody(backlog.AppendTask(item.Body(), text))
	item.SetModified(utils.GetCurrentTimestamp())
	if err := item.Save(); err != nil {
		return err
	}
	fmt.Printf("%s: task added\n", filepath.Base(path))
	return nil
}

func taskTickAction(ctx context.Context, c *cli.Command) error {
	if c.NArg() != 2 {
		fmt.Println("usage: am task tick ITEM_PATH INDEX")
		return nil
	}
	path := mustItemPath(c.Args().Get(0))
	idx, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return fmt.Errorf("index must be an integer: %w", err)
	}
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	body, err := backlog.SetTaskDone(item.Body(), idx, !c.Bool("undo"))
	if err != nil {
		return err
	}
	item.SetBody(body)
	item.SetModified(utils.GetCurrentTimestamp())
	if err := item.Save(); err != nil {
		return err
	}
	state := "ticked"
	if c.Bool("undo") {
		state = "unticked"
	}
	fmt.Printf("%s: task %d %s\n", filepath.Base(path), idx, state)
	return nil
}

func mustItemPath(arg string) string {
	p, _ := filepath.Abs(arg)
	if !strings.HasSuffix(p, ".md") {
		p += ".md"
	}
	return p
}
